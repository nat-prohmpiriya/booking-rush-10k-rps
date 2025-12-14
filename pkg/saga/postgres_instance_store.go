package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store interface using PostgreSQL for saga instances
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL-based saga store
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Save persists a new saga instance
func (s *PostgresStore) Save(ctx context.Context, instance *Instance) error {
	dataJSON, err := json.Marshal(instance.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	stepResultsJSON, err := json.Marshal(instance.StepResults)
	if err != nil {
		return fmt.Errorf("failed to marshal step results: %w", err)
	}

	query := `
		INSERT INTO saga_instances (
			id, definition_id, status, data, step_results,
			current_step, error, created_at, updated_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var errorMsg *string
	if instance.Error != "" {
		errorMsg = &instance.Error
	}

	_, err = s.pool.Exec(ctx, query,
		instance.ID,
		instance.DefinitionID,
		string(instance.Status),
		dataJSON,
		stepResultsJSON,
		instance.CurrentStep,
		errorMsg,
		instance.CreatedAt,
		instance.UpdatedAt,
		instance.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save saga instance: %w", err)
	}

	return nil
}

// Get retrieves a saga instance by ID
func (s *PostgresStore) Get(ctx context.Context, id string) (*Instance, error) {
	query := `
		SELECT id, definition_id, status, data, step_results,
			   current_step, error, created_at, updated_at, completed_at
		FROM saga_instances
		WHERE id = $1
	`

	return s.scanInstance(ctx, s.pool.QueryRow(ctx, query, id))
}

// Update updates an existing saga instance
func (s *PostgresStore) Update(ctx context.Context, instance *Instance) error {
	dataJSON, err := json.Marshal(instance.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	stepResultsJSON, err := json.Marshal(instance.StepResults)
	if err != nil {
		return fmt.Errorf("failed to marshal step results: %w", err)
	}

	query := `
		UPDATE saga_instances
		SET status = $2,
			data = $3,
			step_results = $4,
			current_step = $5,
			error = $6,
			updated_at = $7,
			completed_at = $8
		WHERE id = $1
	`

	var errorMsg *string
	if instance.Error != "" {
		errorMsg = &instance.Error
	}

	result, err := s.pool.Exec(ctx, query,
		instance.ID,
		string(instance.Status),
		dataJSON,
		stepResultsJSON,
		instance.CurrentStep,
		errorMsg,
		time.Now(),
		instance.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update saga instance: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSagaNotFound
	}

	return nil
}

// Delete removes a saga instance
func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM saga_instances WHERE id = $1`

	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete saga instance: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrSagaNotFound
	}

	return nil
}

// GetByStatus retrieves saga instances by status
func (s *PostgresStore) GetByStatus(ctx context.Context, status Status, limit int) ([]*Instance, error) {
	query := `
		SELECT id, definition_id, status, data, step_results,
			   current_step, error, created_at, updated_at, completed_at
		FROM saga_instances
		WHERE status = $1
		ORDER BY created_at ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query, string(status))
	if err != nil {
		return nil, fmt.Errorf("failed to get sagas by status: %w", err)
	}
	defer rows.Close()

	return s.scanInstances(rows)
}

// GetPendingCompensations returns sagas that need compensation
func (s *PostgresStore) GetPendingCompensations(ctx context.Context, limit int) ([]*Instance, error) {
	query := `
		SELECT id, definition_id, status, data, step_results,
			   current_step, error, created_at, updated_at, completed_at
		FROM saga_instances
		WHERE status IN ($1, $2)
		ORDER BY created_at ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query, string(StatusFailed), string(StatusCompensating))
	if err != nil {
		return nil, fmt.Errorf("failed to get pending compensations: %w", err)
	}
	defer rows.Close()

	return s.scanInstances(rows)
}

// GetByDefinitionID retrieves saga instances by definition ID
func (s *PostgresStore) GetByDefinitionID(ctx context.Context, definitionID string, limit int) ([]*Instance, error) {
	query := `
		SELECT id, definition_id, status, data, step_results,
			   current_step, error, created_at, updated_at, completed_at
		FROM saga_instances
		WHERE definition_id = $1
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query, definitionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sagas by definition ID: %w", err)
	}
	defer rows.Close()

	return s.scanInstances(rows)
}

// SaveTransition records a state transition for audit trail
func (s *PostgresStore) SaveTransition(ctx context.Context, sagaID string, fromStatus, toStatus Status, stepName, reason string) error {
	query := `
		INSERT INTO saga_transitions (id, saga_id, from_status, to_status, step_name, reason, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW())
	`

	var stepNamePtr, reasonPtr *string
	if stepName != "" {
		stepNamePtr = &stepName
	}
	if reason != "" {
		reasonPtr = &reason
	}

	_, err := s.pool.Exec(ctx, query, sagaID, string(fromStatus), string(toStatus), stepNamePtr, reasonPtr)
	if err != nil {
		return fmt.Errorf("failed to save transition: %w", err)
	}

	return nil
}

// scanInstance scans a single row into an Instance
func (s *PostgresStore) scanInstance(ctx context.Context, row pgx.Row) (*Instance, error) {
	var instance Instance
	var statusStr string
	var dataJSON, stepResultsJSON []byte
	var errorMsg *string

	err := row.Scan(
		&instance.ID,
		&instance.DefinitionID,
		&statusStr,
		&dataJSON,
		&stepResultsJSON,
		&instance.CurrentStep,
		&errorMsg,
		&instance.CreatedAt,
		&instance.UpdatedAt,
		&instance.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSagaNotFound
		}
		return nil, fmt.Errorf("failed to scan saga instance: %w", err)
	}

	instance.Status = Status(statusStr)

	if errorMsg != nil {
		instance.Error = *errorMsg
	}

	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &instance.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}
	} else {
		instance.Data = make(map[string]interface{})
	}

	if len(stepResultsJSON) > 0 {
		if err := json.Unmarshal(stepResultsJSON, &instance.StepResults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal step results: %w", err)
		}
	} else {
		instance.StepResults = make([]*StepResult, 0)
	}

	return &instance, nil
}

// scanInstances scans multiple rows into a slice of Instances
func (s *PostgresStore) scanInstances(rows pgx.Rows) ([]*Instance, error) {
	var instances []*Instance

	for rows.Next() {
		var instance Instance
		var statusStr string
		var dataJSON, stepResultsJSON []byte
		var errorMsg *string

		err := rows.Scan(
			&instance.ID,
			&instance.DefinitionID,
			&statusStr,
			&dataJSON,
			&stepResultsJSON,
			&instance.CurrentStep,
			&errorMsg,
			&instance.CreatedAt,
			&instance.UpdatedAt,
			&instance.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan saga instance: %w", err)
		}

		instance.Status = Status(statusStr)

		if errorMsg != nil {
			instance.Error = *errorMsg
		}

		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &instance.Data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal data: %w", err)
			}
		} else {
			instance.Data = make(map[string]interface{})
		}

		if len(stepResultsJSON) > 0 {
			if err := json.Unmarshal(stepResultsJSON, &instance.StepResults); err != nil {
				return nil, fmt.Errorf("failed to unmarshal step results: %w", err)
			}
		} else {
			instance.StepResults = make([]*StepResult, 0)
		}

		instances = append(instances, &instance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating saga instances: %w", err)
	}

	return instances, nil
}

// DeadLetter represents a message in the dead letter queue
type DeadLetter struct {
	ID           string                 `json:"id"`
	SagaID       string                 `json:"saga_id,omitempty"`
	Topic        string                 `json:"topic"`
	MessageKey   string                 `json:"message_key,omitempty"`
	MessageValue map[string]interface{} `json:"message_value"`
	ErrorMessage string                 `json:"error_message"`
	RetryCount   int                    `json:"retry_count"`
	CreatedAt    time.Time              `json:"created_at"`
	ProcessedAt  *time.Time             `json:"processed_at,omitempty"`
	Processed    bool                   `json:"processed"`
}

// SaveDeadLetter saves a message to the dead letter queue
func (s *PostgresStore) SaveDeadLetter(ctx context.Context, dl *DeadLetter) error {
	messageJSON, err := json.Marshal(dl.MessageValue)
	if err != nil {
		return fmt.Errorf("failed to marshal message value: %w", err)
	}

	query := `
		INSERT INTO saga_dead_letters (
			saga_id, topic, message_key, message_value, error_message, retry_count
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	var sagaID, messageKey *string
	if dl.SagaID != "" {
		sagaID = &dl.SagaID
	}
	if dl.MessageKey != "" {
		messageKey = &dl.MessageKey
	}

	_, err = s.pool.Exec(ctx, query, sagaID, dl.Topic, messageKey, messageJSON, dl.ErrorMessage, dl.RetryCount)
	if err != nil {
		return fmt.Errorf("failed to save dead letter: %w", err)
	}

	return nil
}

// GetUnprocessedDeadLetters retrieves unprocessed dead letters
func (s *PostgresStore) GetUnprocessedDeadLetters(ctx context.Context, limit int) ([]*DeadLetter, error) {
	query := `
		SELECT id, saga_id, topic, message_key, message_value, error_message, retry_count, created_at
		FROM saga_dead_letters
		WHERE processed = FALSE
		ORDER BY created_at ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get unprocessed dead letters: %w", err)
	}
	defer rows.Close()

	var deadLetters []*DeadLetter
	for rows.Next() {
		var dl DeadLetter
		var sagaID, messageKey *string
		var messageJSON []byte

		err := rows.Scan(
			&dl.ID,
			&sagaID,
			&dl.Topic,
			&messageKey,
			&messageJSON,
			&dl.ErrorMessage,
			&dl.RetryCount,
			&dl.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dead letter: %w", err)
		}

		if sagaID != nil {
			dl.SagaID = *sagaID
		}
		if messageKey != nil {
			dl.MessageKey = *messageKey
		}

		if len(messageJSON) > 0 {
			if err := json.Unmarshal(messageJSON, &dl.MessageValue); err != nil {
				return nil, fmt.Errorf("failed to unmarshal message value: %w", err)
			}
		}

		deadLetters = append(deadLetters, &dl)
	}

	return deadLetters, nil
}

// MarkDeadLetterProcessed marks a dead letter as processed
func (s *PostgresStore) MarkDeadLetterProcessed(ctx context.Context, id string) error {
	query := `
		UPDATE saga_dead_letters
		SET processed = TRUE, processed_at = NOW()
		WHERE id = $1
	`

	_, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark dead letter as processed: %w", err)
	}

	return nil
}
