package database

import (
	"context"
	"fmt"
)

// Transaction management - start, commit, rollback

func (ctx *DatabaseBDDTestContext) iStartADatabaseTransaction() error {
	if ctx.service == nil {
		return fmt.Errorf("no database service available")
	}

	// Start a transaction
	tx, err := ctx.service.Begin()
	if err != nil {
		ctx.lastError = err
		return nil
	}
	ctx.transaction = tx
	return nil
}

func (ctx *DatabaseBDDTestContext) iShouldBeAbleToExecuteQueriesWithinTheTransaction() error {
	if ctx.transaction == nil {
		return fmt.Errorf("no transaction started")
	}

	// Execute query within transaction
	_, err := ctx.transaction.Query("SELECT 1")
	if err != nil {
		ctx.lastError = err
		return nil
	}
	return nil
}

func (ctx *DatabaseBDDTestContext) iShouldBeAbleToCommitOrRollbackTheTransaction() error {
	if ctx.transaction == nil {
		return fmt.Errorf("no transaction to commit/rollback")
	}

	// Try to commit transaction
	err := ctx.transaction.Commit()
	if err != nil {
		ctx.lastError = err
		return nil
	}
	ctx.transaction = nil // Clear transaction after commit
	return nil
}

// Transaction event-related steps

func (ctx *DatabaseBDDTestContext) iHaveStartedADatabaseTransaction() error {
	if ctx.service == nil {
		return fmt.Errorf("no database service available")
	}

	// Reset event observer to capture only this scenario's events
	ctx.eventObserver.Reset()

	// Set the database module as the event emitter for the service
	ctx.service.SetEventEmitter(ctx.module)

	tx, err := ctx.service.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	ctx.transaction = tx
	return nil
}

func (ctx *DatabaseBDDTestContext) iCommitTheTransactionSuccessfully() error {
	if ctx.transaction == nil {
		return fmt.Errorf("no transaction available to commit")
	}

	// Use the real service method to commit transaction and emit events
	err := ctx.service.CommitTransaction(context.Background(), ctx.transaction)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (ctx *DatabaseBDDTestContext) iRollbackTheTransaction() error {
	if ctx.transaction == nil {
		return fmt.Errorf("no transaction available to rollback")
	}

	// Use the real service method to rollback transaction and emit events
	err := ctx.service.RollbackTransaction(context.Background(), ctx.transaction)
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}
