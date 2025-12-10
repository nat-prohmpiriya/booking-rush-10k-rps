package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/logger"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// InventoryWorkerConfig holds configuration for the inventory worker
type InventoryWorkerConfig struct {
	BatchInterval    time.Duration
	MaxBatchSize     int
	RebuildOnStartup bool
}

// ZoneInventoryDelta tracks changes to a zone's inventory
type ZoneInventoryDelta struct {
	ZoneID         string
	ReservedDelta  int // positive = seats reserved, negative = seats released
	ConfirmedDelta int // positive = seats confirmed
	CancelledDelta int // positive = seats cancelled (released back)
}

// InventoryWorker consumes booking events and syncs inventory to PostgreSQL
type InventoryWorker struct {
	config   *InventoryWorkerConfig
	consumer *kafka.Consumer
	db       *database.PostgresDB
	redis    *pkgredis.Client
	log      *logger.Logger

	// Batch aggregation
	mu     sync.Mutex
	deltas map[string]*ZoneInventoryDelta
}

// NewInventoryWorker creates a new inventory worker
func NewInventoryWorker(
	cfg *InventoryWorkerConfig,
	consumer *kafka.Consumer,
	db *database.PostgresDB,
	redis *pkgredis.Client,
	log *logger.Logger,
) *InventoryWorker {
	if cfg.BatchInterval <= 0 {
		cfg.BatchInterval = 5 * time.Second
	}
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = 1000
	}

	return &InventoryWorker{
		config:   cfg,
		consumer: consumer,
		db:       db,
		redis:    redis,
		log:      log,
		deltas:   make(map[string]*ZoneInventoryDelta),
	}
}

// Start begins consuming events and syncing inventory
func (w *InventoryWorker) Start(ctx context.Context) {
	// Start batch flush ticker
	ticker := time.NewTicker(w.config.BatchInterval)
	defer ticker.Stop()

	// Channel to trigger batch flush
	flushCh := make(chan struct{}, 1)

	// Start consumer loop
	go w.consumeLoop(ctx, flushCh)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Inventory worker context cancelled, flushing remaining batch...")
			w.flushBatch(context.Background())
			return
		case <-ticker.C:
			w.flushBatch(ctx)
		case <-flushCh:
			w.flushBatch(ctx)
		}
	}
}

// consumeLoop continuously polls for new events
func (w *InventoryWorker) consumeLoop(ctx context.Context, flushCh chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			records, err := w.consumer.Poll(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				w.log.Error(fmt.Sprintf("Failed to poll Kafka: %v", err))
				time.Sleep(time.Second)
				continue
			}

			if len(records) == 0 {
				continue
			}

			w.processRecords(ctx, records)

			// Commit offsets after processing
			if err := w.consumer.CommitRecords(ctx, records); err != nil {
				w.log.Error(fmt.Sprintf("Failed to commit offsets: %v", err))
			}

			// Check if batch size exceeded
			w.mu.Lock()
			batchSize := len(w.deltas)
			w.mu.Unlock()

			if batchSize >= w.config.MaxBatchSize {
				select {
				case flushCh <- struct{}{}:
				default:
				}
			}
		}
	}
}

// processRecords processes a batch of Kafka records
func (w *InventoryWorker) processRecords(_ context.Context, records []*kafka.Record) {
	for _, record := range records {
		if err := w.processRecord(record); err != nil {
			w.log.Error(fmt.Sprintf("Failed to process record: %v", err))
		}
	}
}

// processRecord processes a single Kafka record
func (w *InventoryWorker) processRecord(record *kafka.Record) error {
	var event domain.BookingEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal booking event: %w", err)
	}

	if event.BookingData == nil {
		return fmt.Errorf("booking event has no data")
	}

	w.aggregateDelta(&event)
	return nil
}

// aggregateDelta aggregates the inventory delta for a zone
func (w *InventoryWorker) aggregateDelta(event *domain.BookingEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	zoneID := event.BookingData.ZoneID
	quantity := event.BookingData.Quantity

	delta, exists := w.deltas[zoneID]
	if !exists {
		delta = &ZoneInventoryDelta{ZoneID: zoneID}
		w.deltas[zoneID] = delta
	}

	switch event.EventType {
	case domain.BookingEventCreated:
		// Seats reserved: decrease available, increase reserved
		delta.ReservedDelta += quantity
	case domain.BookingEventConfirmed:
		// Seats confirmed: move from reserved to sold
		delta.ConfirmedDelta += quantity
	case domain.BookingEventCancelled, domain.BookingEventExpired:
		// Seats released: decrease reserved, increase available
		delta.CancelledDelta += quantity
	}
}

// flushBatch writes aggregated deltas to PostgreSQL
func (w *InventoryWorker) flushBatch(ctx context.Context) {
	w.mu.Lock()
	if len(w.deltas) == 0 {
		w.mu.Unlock()
		return
	}

	// Swap out the deltas map
	deltas := w.deltas
	w.deltas = make(map[string]*ZoneInventoryDelta)
	w.mu.Unlock()

	w.log.Info(fmt.Sprintf("Flushing batch with %d zone updates", len(deltas)))

	// Begin transaction
	tx, err := w.db.BeginTx(ctx)
	if err != nil {
		w.log.Error(fmt.Sprintf("Failed to begin transaction: %v", err))
		// Put deltas back for retry
		w.restoreDeltas(deltas)
		return
	}

	// Update each zone
	for zoneID, delta := range deltas {
		if err := w.updateZoneInventory(ctx, tx, delta); err != nil {
			w.log.Error(fmt.Sprintf("Failed to update zone %s: %v", zoneID, err))
			tx.Rollback(ctx)
			w.restoreDeltas(deltas)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		w.log.Error(fmt.Sprintf("Failed to commit transaction: %v", err))
		w.restoreDeltas(deltas)
		return
	}

	w.log.Info(fmt.Sprintf("Successfully synced %d zones to PostgreSQL", len(deltas)))
}

// updateZoneInventory updates a single zone's inventory in PostgreSQL
func (w *InventoryWorker) updateZoneInventory(ctx context.Context, tx pgx.Tx, delta *ZoneInventoryDelta) error {
	// Calculate net changes:
	// - ReservedDelta: seats that got reserved (decrease available)
	// - ConfirmedDelta: seats that moved from reserved to sold
	// - CancelledDelta: seats that got released (increase available)

	// available_seats change: -reserved + cancelled
	// reserved_seats change: +reserved - confirmed - cancelled
	// sold_seats change: +confirmed

	availableChange := -delta.ReservedDelta + delta.CancelledDelta
	reservedChange := delta.ReservedDelta - delta.ConfirmedDelta - delta.CancelledDelta
	soldChange := delta.ConfirmedDelta

	query := `
		UPDATE seat_zones
		SET
			available_seats = available_seats + $1,
			reserved_seats = reserved_seats + $2,
			sold_seats = sold_seats + $3,
			updated_at = NOW()
		WHERE id = $4
	`

	_, err := tx.Exec(ctx, query, availableChange, reservedChange, soldChange, delta.ZoneID)
	return err
}

// restoreDeltas puts deltas back for retry
func (w *InventoryWorker) restoreDeltas(deltas map[string]*ZoneInventoryDelta) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for zoneID, delta := range deltas {
		if existing, ok := w.deltas[zoneID]; ok {
			// Merge with any new deltas
			existing.ReservedDelta += delta.ReservedDelta
			existing.ConfirmedDelta += delta.ConfirmedDelta
			existing.CancelledDelta += delta.CancelledDelta
		} else {
			w.deltas[zoneID] = delta
		}
	}
}

// RebuildRedisFromDB rebuilds Redis inventory from PostgreSQL
func (w *InventoryWorker) RebuildRedisFromDB(ctx context.Context) error {
	w.log.Info("Starting Redis rebuild from PostgreSQL...")

	// Query all active seat zones
	query := `
		SELECT id, available_seats
		FROM seat_zones
		WHERE is_active = true AND deleted_at IS NULL
	`

	rows, err := w.db.Pool().Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query seat zones: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var zoneID string
		var availableSeats int64

		if err := rows.Scan(&zoneID, &availableSeats); err != nil {
			w.log.Error(fmt.Sprintf("Failed to scan zone row: %v", err))
			continue
		}

		// Set zone availability in Redis
		key := fmt.Sprintf("zone:availability:%s", zoneID)
		if err := w.redis.Set(ctx, key, availableSeats, 0).Err(); err != nil {
			w.log.Error(fmt.Sprintf("Failed to set Redis key %s: %v", key, err))
			continue
		}

		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	w.log.Info(fmt.Sprintf("Redis rebuild complete: %d zones synced", count))
	return nil
}

// GetPendingDeltaCount returns the number of pending deltas
func (w *InventoryWorker) GetPendingDeltaCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.deltas)
}
