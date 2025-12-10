package worker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/kafka"
)

func TestAggregateDelta_BookingCreated(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	event := &domain.BookingEvent{
		EventType: domain.BookingEventCreated,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 2,
		},
	}

	worker.aggregateDelta(event)

	if len(worker.deltas) != 1 {
		t.Errorf("Expected 1 delta, got %d", len(worker.deltas))
	}

	delta := worker.deltas["zone-1"]
	if delta == nil {
		t.Fatal("Expected delta for zone-1")
	}

	if delta.ReservedDelta != 2 {
		t.Errorf("Expected ReservedDelta=2, got %d", delta.ReservedDelta)
	}
	if delta.ConfirmedDelta != 0 {
		t.Errorf("Expected ConfirmedDelta=0, got %d", delta.ConfirmedDelta)
	}
	if delta.CancelledDelta != 0 {
		t.Errorf("Expected CancelledDelta=0, got %d", delta.CancelledDelta)
	}
}

func TestAggregateDelta_BookingConfirmed(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	event := &domain.BookingEvent{
		EventType: domain.BookingEventConfirmed,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 3,
		},
	}

	worker.aggregateDelta(event)

	delta := worker.deltas["zone-1"]
	if delta == nil {
		t.Fatal("Expected delta for zone-1")
	}

	if delta.ConfirmedDelta != 3 {
		t.Errorf("Expected ConfirmedDelta=3, got %d", delta.ConfirmedDelta)
	}
}

func TestAggregateDelta_BookingCancelled(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	event := &domain.BookingEvent{
		EventType: domain.BookingEventCancelled,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 2,
		},
	}

	worker.aggregateDelta(event)

	delta := worker.deltas["zone-1"]
	if delta == nil {
		t.Fatal("Expected delta for zone-1")
	}

	if delta.CancelledDelta != 2 {
		t.Errorf("Expected CancelledDelta=2, got %d", delta.CancelledDelta)
	}
}

func TestAggregateDelta_BookingExpired(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	event := &domain.BookingEvent{
		EventType: domain.BookingEventExpired,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 1,
		},
	}

	worker.aggregateDelta(event)

	delta := worker.deltas["zone-1"]
	if delta == nil {
		t.Fatal("Expected delta for zone-1")
	}

	if delta.CancelledDelta != 1 {
		t.Errorf("Expected CancelledDelta=1, got %d", delta.CancelledDelta)
	}
}

func TestAggregateDelta_MultipleEventsForSameZone(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	// Reserve 5 seats
	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventCreated,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 5,
		},
	})

	// Confirm 3 seats
	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventConfirmed,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 3,
		},
	})

	// Cancel 2 seats
	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventCancelled,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 2,
		},
	})

	delta := worker.deltas["zone-1"]
	if delta == nil {
		t.Fatal("Expected delta for zone-1")
	}

	if delta.ReservedDelta != 5 {
		t.Errorf("Expected ReservedDelta=5, got %d", delta.ReservedDelta)
	}
	if delta.ConfirmedDelta != 3 {
		t.Errorf("Expected ConfirmedDelta=3, got %d", delta.ConfirmedDelta)
	}
	if delta.CancelledDelta != 2 {
		t.Errorf("Expected CancelledDelta=2, got %d", delta.CancelledDelta)
	}
}

func TestAggregateDelta_MultipleZones(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventCreated,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-1",
			Quantity: 2,
		},
	})

	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventCreated,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-2",
			Quantity: 3,
		},
	})

	if len(worker.deltas) != 2 {
		t.Errorf("Expected 2 deltas, got %d", len(worker.deltas))
	}

	if worker.deltas["zone-1"].ReservedDelta != 2 {
		t.Errorf("Expected zone-1 ReservedDelta=2, got %d", worker.deltas["zone-1"].ReservedDelta)
	}
	if worker.deltas["zone-2"].ReservedDelta != 3 {
		t.Errorf("Expected zone-2 ReservedDelta=3, got %d", worker.deltas["zone-2"].ReservedDelta)
	}
}

func TestProcessRecord_ValidEvent(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	event := domain.BookingEvent{
		EventType: domain.BookingEventCreated,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-test",
			Quantity: 4,
		},
	}

	eventJSON, _ := json.Marshal(event)
	record := &kafka.Record{
		Value: eventJSON,
	}

	err := worker.processRecord(record)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if worker.deltas["zone-test"].ReservedDelta != 4 {
		t.Errorf("Expected ReservedDelta=4, got %d", worker.deltas["zone-test"].ReservedDelta)
	}
}

func TestProcessRecord_InvalidJSON(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	record := &kafka.Record{
		Value: []byte("invalid json"),
	}

	err := worker.processRecord(record)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestProcessRecord_NilBookingData(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	event := domain.BookingEvent{
		EventType:   domain.BookingEventCreated,
		BookingData: nil,
	}

	eventJSON, _ := json.Marshal(event)
	record := &kafka.Record{
		Value: eventJSON,
	}

	err := worker.processRecord(record)
	if err == nil {
		t.Error("Expected error for nil booking data")
	}
}

func TestRestoreDeltas(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	// Add initial delta
	worker.deltas["zone-1"] = &ZoneInventoryDelta{
		ZoneID:        "zone-1",
		ReservedDelta: 2,
	}

	// Restore some deltas
	oldDeltas := map[string]*ZoneInventoryDelta{
		"zone-1": {
			ZoneID:        "zone-1",
			ReservedDelta: 3,
		},
		"zone-2": {
			ZoneID:        "zone-2",
			ReservedDelta: 5,
		},
	}

	worker.restoreDeltas(oldDeltas)

	// zone-1 should be merged
	if worker.deltas["zone-1"].ReservedDelta != 5 {
		t.Errorf("Expected zone-1 ReservedDelta=5, got %d", worker.deltas["zone-1"].ReservedDelta)
	}

	// zone-2 should be added
	if worker.deltas["zone-2"].ReservedDelta != 5 {
		t.Errorf("Expected zone-2 ReservedDelta=5, got %d", worker.deltas["zone-2"].ReservedDelta)
	}
}

func TestGetPendingDeltaCount(t *testing.T) {
	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	if worker.GetPendingDeltaCount() != 0 {
		t.Errorf("Expected 0 pending deltas, got %d", worker.GetPendingDeltaCount())
	}

	worker.deltas["zone-1"] = &ZoneInventoryDelta{ZoneID: "zone-1"}
	worker.deltas["zone-2"] = &ZoneInventoryDelta{ZoneID: "zone-2"}

	if worker.GetPendingDeltaCount() != 2 {
		t.Errorf("Expected 2 pending deltas, got %d", worker.GetPendingDeltaCount())
	}
}

func TestInventoryDeltaCalculation(t *testing.T) {
	// Test the inventory delta calculation logic
	// Scenario: Zone starts with 100 available, 0 reserved, 0 sold
	// Events:
	// 1. Reserve 10 seats (reserved: 10, available: -10)
	// 2. Reserve 5 seats (reserved: 5, available: -5)
	// 3. Confirm 8 seats (confirmed: 8)
	// 4. Cancel 3 seats (cancelled: 3)

	worker := &InventoryWorker{
		config: &InventoryWorkerConfig{
			BatchInterval: 5 * time.Second,
			MaxBatchSize:  100,
		},
		deltas: make(map[string]*ZoneInventoryDelta),
	}

	// Reserve 10
	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventCreated,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-calc",
			Quantity: 10,
		},
	})

	// Reserve 5
	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventCreated,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-calc",
			Quantity: 5,
		},
	})

	// Confirm 8
	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventConfirmed,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-calc",
			Quantity: 8,
		},
	})

	// Cancel 3
	worker.aggregateDelta(&domain.BookingEvent{
		EventType: domain.BookingEventCancelled,
		BookingData: &domain.BookingEventData{
			ZoneID:   "zone-calc",
			Quantity: 3,
		},
	})

	delta := worker.deltas["zone-calc"]

	// Verify deltas
	if delta.ReservedDelta != 15 {
		t.Errorf("Expected ReservedDelta=15, got %d", delta.ReservedDelta)
	}
	if delta.ConfirmedDelta != 8 {
		t.Errorf("Expected ConfirmedDelta=8, got %d", delta.ConfirmedDelta)
	}
	if delta.CancelledDelta != 3 {
		t.Errorf("Expected CancelledDelta=3, got %d", delta.CancelledDelta)
	}

	// Calculate expected changes:
	// available_seats change: -reserved + cancelled = -15 + 3 = -12
	// reserved_seats change: +reserved - confirmed - cancelled = 15 - 8 - 3 = 4
	// sold_seats change: +confirmed = 8

	availableChange := -delta.ReservedDelta + delta.CancelledDelta
	reservedChange := delta.ReservedDelta - delta.ConfirmedDelta - delta.CancelledDelta
	soldChange := delta.ConfirmedDelta

	if availableChange != -12 {
		t.Errorf("Expected availableChange=-12, got %d", availableChange)
	}
	if reservedChange != 4 {
		t.Errorf("Expected reservedChange=4, got %d", reservedChange)
	}
	if soldChange != 8 {
		t.Errorf("Expected soldChange=8, got %d", soldChange)
	}

	// Verify: Initial state: available=100, reserved=0, sold=0
	// Final state: available=88, reserved=4, sold=8
	// Total should still be 100: 88 + 4 + 8 = 100
	initialAvailable := 100
	finalAvailable := initialAvailable + availableChange
	finalReserved := 0 + reservedChange
	finalSold := 0 + soldChange

	total := finalAvailable + finalReserved + finalSold
	if total != 100 {
		t.Errorf("Inventory not balanced: available=%d, reserved=%d, sold=%d, total=%d",
			finalAvailable, finalReserved, finalSold, total)
	}
}
