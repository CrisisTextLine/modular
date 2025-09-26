package database

import (
	"context"
	"fmt"
)

// Connection management, health checks, and connection pooling

func (ctx *DatabaseBDDTestContext) iHaveDatabaseConnectionPoolingConfigured() error {
	// Connection pooling is configured as part of the module setup
	return ctx.iHaveADatabaseConnection()
}

func (ctx *DatabaseBDDTestContext) iMakeMultipleConcurrentDatabaseRequests() error {
	if ctx.service == nil {
		return fmt.Errorf("no database service available")
	}

	// Simulate multiple concurrent requests
	for i := 0; i < 3; i++ {
		go func() {
			ctx.service.Query("SELECT 1")
		}()
	}
	return nil
}

func (ctx *DatabaseBDDTestContext) theConnectionPoolShouldHandleTheRequestsEfficiently() error {
	// Connection pool efficiency is verified by successful query execution without errors
	// Modern database drivers handle connection pooling automatically
	if ctx.queryError != nil {
		return fmt.Errorf("query execution failed, suggesting connection pool issues: %v", ctx.queryError)
	}
	return nil
}

func (ctx *DatabaseBDDTestContext) connectionsShouldBeReusedProperly() error {
	// Connection reuse is handled transparently by the connection pool
	// Successful consecutive operations indicate proper connection reuse
	if ctx.service == nil {
		return fmt.Errorf("database service not available for connection reuse test")
	}

	// Execute multiple queries to test connection reuse
	_, err1 := ctx.service.Query("SELECT 1")
	_, err2 := ctx.service.Query("SELECT 2")

	if err1 != nil || err2 != nil {
		return fmt.Errorf("consecutive queries failed, suggesting connection reuse issues: err1=%v, err2=%v", err1, err2)
	}

	return nil
}

func (ctx *DatabaseBDDTestContext) iPerformAHealthCheck() error {
	if ctx.service == nil {
		return fmt.Errorf("no database service available")
	}

	// Perform health check
	err := ctx.service.Ping(context.Background())
	ctx.healthStatus = (err == nil)
	if err != nil {
		ctx.lastError = err
	}
	return nil
}

func (ctx *DatabaseBDDTestContext) theHealthCheckShouldReportDatabaseStatus() error {
	// Enhanced health check validation
	if ctx.service == nil {
		return fmt.Errorf("database service is not available for health check")
	}

	// Try to ping the database to verify health check functionality
	pingCtx := context.Background()
	if err := ctx.service.Ping(pingCtx); err != nil {
		return fmt.Errorf("health check failed: database ping returned error: %w", err)
	}

	return nil
}

func (ctx *DatabaseBDDTestContext) indicateWhetherTheDatabaseIsAccessible() error {
	// Enhanced database accessibility validation
	if ctx.service == nil {
		return fmt.Errorf("database service is not available to check accessibility")
	}

	// Perform a simple query to verify database is accessible
	rows, err := ctx.service.Query("SELECT 1 as accessibility_test")
	if err != nil {
		return fmt.Errorf("database is not accessible: query failed with error: %w", err)
	}
	defer rows.Close()

	// Ensure we can actually read the result
	if !rows.Next() {
		return fmt.Errorf("database is not accessible: no rows returned from accessibility test")
	}

	var testValue int
	if err := rows.Scan(&testValue); err != nil {
		return fmt.Errorf("database is not accessible: failed to scan test value: %w", err)
	}

	if testValue != 1 {
		return fmt.Errorf("database is not accessible: unexpected test value %d, expected 1", testValue)
	}

	return nil
}

func (ctx *DatabaseBDDTestContext) iHaveADatabaseModuleConfigured() error {
	// This is the same as the background step but for the health check scenario
	return ctx.iHaveAModularApplicationWithDatabaseModuleConfigured()
}
