package database

import (
	"context"
	"fmt"
	"time"
)

// Database migration events and handling

func (ctx *DatabaseBDDTestContext) aDatabaseMigrationIsInitiated() error {
	// Reset event observer to capture only this scenario's events
	ctx.eventObserver.Reset()

	// Create a simple test migration
	migration := Migration{
		ID:      "test-migration-001",
		Version: "1.0.0",
		SQL:     "CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY, name TEXT)",
		Up:      true,
	}

	// Get the database service and set up event emission
	if ctx.service != nil {
		// Set the database module as the event emitter for the service
		ctx.service.SetEventEmitter(ctx.module)

		// Create migrations table first
		err := ctx.service.CreateMigrationsTable(context.Background())
		if err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}

		// Run the migration - this should emit the migration started event
		err = ctx.service.RunMigration(context.Background(), migration)
		if err != nil {
			ctx.lastError = err
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (ctx *DatabaseBDDTestContext) aMigrationStartedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMigrationStarted {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeMigrationStarted, eventTypes)
}

func (ctx *DatabaseBDDTestContext) theEventShouldContainMigrationMetadata() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMigrationStarted {
			// Check that the event has migration metadata
			data := event.Data()
			if data == nil {
				return fmt.Errorf("migration started event should contain metadata but data is nil")
			}
			return nil
		}
	}
	return fmt.Errorf("migration started event not found to validate metadata")
}

func (ctx *DatabaseBDDTestContext) aDatabaseMigrationCompletesSuccessfully() error {
	// Reset event observer to capture only this scenario's events
	ctx.eventObserver.Reset()

	// Create a test migration that will complete successfully
	migration := Migration{
		ID:      "test-migration-002",
		Version: "1.1.0",
		SQL:     "CREATE TABLE IF NOT EXISTS completed_table (id INTEGER PRIMARY KEY, status TEXT DEFAULT 'completed')",
		Up:      true,
	}

	// Get the database service and set up event emission
	if ctx.service != nil {
		// Set the database module as the event emitter for the service
		ctx.service.SetEventEmitter(ctx.module)

		// Create migrations table first
		err := ctx.service.CreateMigrationsTable(context.Background())
		if err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}

		// Run the migration - this should emit migration started and completed events
		err = ctx.service.RunMigration(context.Background(), migration)
		if err != nil {
			ctx.lastError = err
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (ctx *DatabaseBDDTestContext) aMigrationCompletedEventShouldBeEmitted() error {
	time.Sleep(100 * time.Millisecond) // Give time for async event emission

	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMigrationCompleted {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeMigrationCompleted, eventTypes)
}

func (ctx *DatabaseBDDTestContext) theEventShouldContainMigrationResults() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMigrationCompleted {
			// Check that the event has migration results
			data := event.Data()
			if data == nil {
				return fmt.Errorf("migration completed event should contain results but data is nil")
			}
			return nil
		}
	}
	return fmt.Errorf("migration completed event not found to validate results")
}

func (ctx *DatabaseBDDTestContext) aDatabaseMigrationFailsWithErrors() error {
	// Reset event observer to capture only this scenario's events
	ctx.eventObserver.Reset()

	// Create a migration with invalid SQL that will fail
	migration := Migration{
		ID:      "test-migration-fail",
		Version: "1.2.0",
		SQL:     "CREATE TABLE duplicate_table (id INTEGER PRIMARY KEY); CREATE TABLE duplicate_table (name TEXT);", // This will fail due to duplicate table
		Up:      true,
	}

	// Get the database service and set up event emission
	if ctx.service != nil {
		// Set the database module as the event emitter for the service
		ctx.service.SetEventEmitter(ctx.module)

		// Run the migration - this should fail and emit migration failed event
		err := ctx.service.RunMigration(context.Background(), migration)
		if err != nil {
			// This is expected - the migration should fail
			ctx.lastError = err
		}
	}

	return nil
}

func (ctx *DatabaseBDDTestContext) aMigrationFailedEventShouldBeEmitted() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMigrationFailed {
			return nil
		}
	}

	eventTypes := make([]string, len(events))
	for i, event := range events {
		eventTypes[i] = event.Type()
	}

	return fmt.Errorf("event of type %s was not emitted. Captured events: %v", EventTypeMigrationFailed, eventTypes)
}

func (ctx *DatabaseBDDTestContext) theEventShouldContainFailureDetails() error {
	events := ctx.eventObserver.GetEvents()
	for _, event := range events {
		if event.Type() == EventTypeMigrationFailed {
			// Check that the event has failure details
			data := event.Data()
			if data == nil {
				return fmt.Errorf("migration failed event should contain failure details but data is nil")
			}
			return nil
		}
	}
	return fmt.Errorf("migration failed event not found to validate failure details")
}
