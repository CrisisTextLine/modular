package database

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
	
	"github.com/CrisisTextLine/modular"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Migration represents a database migration
type Migration struct {
	ID      string
	Version string
	SQL     string
	Up      bool // true for up migration, false for down
}

// MigrationService provides migration functionality
type MigrationService interface {
	// RunMigration executes a single migration
	RunMigration(ctx context.Context, migration Migration) error
	
	// GetAppliedMigrations returns a list of already applied migrations
	GetAppliedMigrations(ctx context.Context) ([]string, error)
	
	// CreateMigrationsTable creates the migrations tracking table
	CreateMigrationsTable(ctx context.Context) error
}

// migrationServiceImpl implements MigrationService
type migrationServiceImpl struct {
	db            *sql.DB
	eventEmitter  EventEmitter
	tableName     string
}

// EventEmitter interface for emitting migration events
type EventEmitter interface {
	// EmitEvent emits a cloud event with the provided context
	EmitEvent(ctx context.Context, event cloudevents.Event) error
}

// NewMigrationService creates a new migration service
func NewMigrationService(db *sql.DB, eventEmitter EventEmitter) MigrationService {
	return &migrationServiceImpl{
		db:           db,
		eventEmitter: eventEmitter,
		tableName:    "schema_migrations",
	}
}

// CreateMigrationsTable creates the migrations tracking table if it doesn't exist
func (m *migrationServiceImpl) CreateMigrationsTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			version TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`, m.tableName)

	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns a list of migration IDs that have been applied
func (m *migrationServiceImpl) GetAppliedMigrations(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("SELECT id FROM %s ORDER BY applied_at", m.tableName)
	
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		migrations = append(migrations, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return migrations, nil
}

// RunMigration executes a migration and tracks it
func (m *migrationServiceImpl) RunMigration(ctx context.Context, migration Migration) error {
	startTime := time.Now()

	// Emit migration started event
	if m.eventEmitter != nil {
		event := modular.NewCloudEvent(EventTypeMigrationStarted, "database-migration", map[string]interface{}{
			"migration_id": migration.ID,
			"version":      migration.Version,
		}, nil)
		m.eventEmitter.EmitEvent(ctx, event)
	}

	// Start a transaction for the migration
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		// Emit migration failed event
		if m.eventEmitter != nil {
			event := modular.NewCloudEvent(EventTypeMigrationFailed, "database-migration", map[string]interface{}{
				"migration_id": migration.ID,
				"version":      migration.Version,
				"error":        err.Error(),
				"duration_ms":  time.Since(startTime).Milliseconds(),
			}, nil)
			m.eventEmitter.EmitEvent(ctx, event)
		}
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Execute the migration SQL
	_, err = tx.ExecContext(ctx, migration.SQL)
	if err != nil {
		// Emit migration failed event
		if m.eventEmitter != nil {
			event := modular.NewCloudEvent(EventTypeMigrationFailed, "database-migration", map[string]interface{}{
				"migration_id": migration.ID,
				"version":      migration.Version,
				"error":        err.Error(),
				"duration_ms":  time.Since(startTime).Milliseconds(),
			}, nil)
			m.eventEmitter.EmitEvent(ctx, event)
		}
		return fmt.Errorf("failed to execute migration %s: %w", migration.ID, err)
	}

	// Record the migration as applied
	recordQuery := fmt.Sprintf("INSERT INTO %s (id, version) VALUES (?, ?)", m.tableName)
	_, err = tx.ExecContext(ctx, recordQuery, migration.ID, migration.Version)
	if err != nil {
		// Emit migration failed event
		if m.eventEmitter != nil {
			event := modular.NewCloudEvent(EventTypeMigrationFailed, "database-migration", map[string]interface{}{
				"migration_id": migration.ID,
				"version":      migration.Version,
				"error":        err.Error(),
				"duration_ms":  time.Since(startTime).Milliseconds(),
			}, nil)
			m.eventEmitter.EmitEvent(ctx, event)
		}
		return fmt.Errorf("failed to record migration %s: %w", migration.ID, err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		// Emit migration failed event
		if m.eventEmitter != nil {
			event := modular.NewCloudEvent(EventTypeMigrationFailed, "database-migration", map[string]interface{}{
				"migration_id": migration.ID,
				"version":      migration.Version,
				"error":        err.Error(),
				"duration_ms":  time.Since(startTime).Milliseconds(),
			}, nil)
			m.eventEmitter.EmitEvent(ctx, event)
		}
		return fmt.Errorf("failed to commit migration %s: %w", migration.ID, err)
	}

	// Emit migration completed event
	if m.eventEmitter != nil {
		event := modular.NewCloudEvent(EventTypeMigrationCompleted, "database-migration", map[string]interface{}{
			"migration_id": migration.ID,
			"version":      migration.Version,
			"duration_ms":  time.Since(startTime).Milliseconds(),
		}, nil)
		m.eventEmitter.EmitEvent(ctx, event)
	}

	return nil
}

// MigrationRunner helps run multiple migrations
type MigrationRunner struct {
	service MigrationService
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(service MigrationService) *MigrationRunner {
	return &MigrationRunner{
		service: service,
	}
}

// RunMigrations runs a set of migrations in order
func (r *MigrationRunner) RunMigrations(ctx context.Context, migrations []Migration) error {
	// Sort migrations by version to ensure correct order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Ensure migrations table exists
	if err := r.service.CreateMigrationsTable(ctx); err != nil {
		return err
	}

	// Get already applied migrations
	applied, err := r.service.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	appliedMap := make(map[string]bool)
	for _, id := range applied {
		appliedMap[id] = true
	}

	// Run pending migrations
	for _, migration := range migrations {
		if !appliedMap[migration.ID] {
			if err := r.service.RunMigration(ctx, migration); err != nil {
				return err
			}
		}
	}

	return nil
}