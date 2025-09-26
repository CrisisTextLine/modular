package database

import (
	"context"

	"github.com/cucumber/godog"
)

// InitializeDatabaseScenario initializes the database BDD test scenario
func InitializeDatabaseScenario(ctx *godog.ScenarioContext) {
	testCtx := &DatabaseBDDTestContext{}

	// Reset context before each scenario
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		testCtx.resetContext()
		return ctx, nil
	})

	// Background steps
	ctx.Step(`^I have a modular application with database module configured$`, testCtx.iHaveAModularApplicationWithDatabaseModuleConfigured)

	// Module initialization steps
	ctx.Step(`^the database module is initialized$`, testCtx.theDatabaseModuleIsInitialized)
	ctx.Step(`^the database service should be available$`, testCtx.theDatabaseServiceShouldBeAvailable)
	ctx.Step(`^database connections should be configured$`, testCtx.databaseConnectionsShouldBeConfigured)

	// Query execution steps
	ctx.Step(`^I have a database connection$`, testCtx.iHaveADatabaseConnection)
	ctx.Step(`^I execute a simple SQL query$`, testCtx.iExecuteASimpleSQLQuery)
	ctx.Step(`^the query should execute successfully$`, testCtx.theQueryShouldExecuteSuccessfully)
	ctx.Step(`^I should receive the expected results$`, testCtx.iShouldReceiveTheExpectedResults)

	// Parameterized query steps
	ctx.Step(`^I execute a parameterized SQL query$`, testCtx.iExecuteAParameterizedSQLQuery)
	ctx.Step(`^the query should execute successfully with parameters$`, testCtx.theQueryShouldExecuteSuccessfullyWithParameters)
	ctx.Step(`^the parameters should be properly escaped$`, testCtx.theParametersShouldBeProperlyEscaped)

	// Error handling steps
	ctx.Step(`^I have an invalid database configuration$`, testCtx.iHaveAnInvalidDatabaseConfiguration)
	ctx.Step(`^I try to execute a query$`, testCtx.iTryToExecuteAQuery)
	ctx.Step(`^the operation should fail gracefully$`, testCtx.theOperationShouldFailGracefully)
	ctx.Step(`^an appropriate database error should be returned$`, testCtx.anAppropriateDatabaseErrorShouldBeReturned)

	// Transaction steps
	ctx.Step(`^I start a database transaction$`, testCtx.iStartADatabaseTransaction)
	ctx.Step(`^I should be able to execute queries within the transaction$`, testCtx.iShouldBeAbleToExecuteQueriesWithinTheTransaction)
	ctx.Step(`^I should be able to commit or rollback the transaction$`, testCtx.iShouldBeAbleToCommitOrRollbackTheTransaction)

	// Connection pool steps
	ctx.Step(`^I have database connection pooling configured$`, testCtx.iHaveDatabaseConnectionPoolingConfigured)
	ctx.Step(`^I make multiple concurrent database requests$`, testCtx.iMakeMultipleConcurrentDatabaseRequests)
	ctx.Step(`^the connection pool should handle the requests efficiently$`, testCtx.theConnectionPoolShouldHandleTheRequestsEfficiently)
	ctx.Step(`^connections should be reused properly$`, testCtx.connectionsShouldBeReusedProperly)

	// Health check steps
	ctx.Step(`^I have a database module configured$`, testCtx.iHaveADatabaseModuleConfigured)
	ctx.Step(`^I perform a health check$`, testCtx.iPerformAHealthCheck)
	ctx.Step(`^the health check should report database status$`, testCtx.theHealthCheckShouldReportDatabaseStatus)
	ctx.Step(`^indicate whether the database is accessible$`, testCtx.indicateWhetherTheDatabaseIsAccessible)

	// Event observation steps
	ctx.Step(`^I have a database service with event observation enabled$`, testCtx.iHaveADatabaseServiceWithEventObservationEnabled)
	ctx.Step(`^I execute a database query$`, testCtx.iExecuteADatabaseQuery)
	ctx.Step(`^a query executed event should be emitted$`, testCtx.aQueryExecutedEventShouldBeEmitted)
	ctx.Step(`^the event should contain query performance metrics$`, testCtx.theEventShouldContainQueryPerformanceMetrics)
	ctx.Step(`^a transaction started event should be emitted$`, testCtx.aTransactionStartedEventShouldBeEmitted)
	ctx.Step(`^the query fails with an error$`, testCtx.theQueryFailsWithAnError)
	ctx.Step(`^a query error event should be emitted$`, testCtx.aQueryErrorEventShouldBeEmitted)
	ctx.Step(`^the event should contain error details$`, testCtx.theEventShouldContainErrorDetails)
	ctx.Step(`^the database module starts$`, testCtx.theDatabaseModuleStarts)
	ctx.Step(`^a configuration loaded event should be emitted$`, testCtx.aConfigurationLoadedEventShouldBeEmitted)
	ctx.Step(`^a database connected event should be emitted$`, testCtx.aDatabaseConnectedEventShouldBeEmitted)
	ctx.Step(`^the database module stops$`, testCtx.theDatabaseModuleStops)
	ctx.Step(`^a database disconnected event should be emitted$`, testCtx.aDatabaseDisconnectedEventShouldBeEmitted)

	// Connection error event steps
	ctx.Step(`^a database connection fails with invalid credentials$`, testCtx.aDatabaseConnectionFailsWithInvalidCredentials)
	ctx.Step(`^a connection error event should be emitted$`, testCtx.aConnectionErrorEventShouldBeEmitted)
	ctx.Step(`^the event should contain connection failure details$`, testCtx.theEventShouldContainConnectionFailureDetails)

	// Transaction commit event steps
	ctx.Step(`^I have started a database transaction$`, testCtx.iHaveStartedADatabaseTransaction)
	ctx.Step(`^I commit the transaction successfully$`, testCtx.iCommitTheTransactionSuccessfully)
	ctx.Step(`^a transaction committed event should be emitted$`, testCtx.aTransactionCommittedEventShouldBeEmitted)
	ctx.Step(`^the event should contain transaction details$`, testCtx.theEventShouldContainTransactionDetails)

	// Transaction rollback event steps
	ctx.Step(`^I rollback the transaction$`, testCtx.iRollbackTheTransaction)
	ctx.Step(`^a transaction rolled back event should be emitted$`, testCtx.aTransactionRolledBackEventShouldBeEmitted)
	ctx.Step(`^the event should contain rollback details$`, testCtx.theEventShouldContainRollbackDetails)

	// Migration event steps
	ctx.Step(`^a database migration is initiated$`, testCtx.aDatabaseMigrationIsInitiated)
	ctx.Step(`^a migration started event should be emitted$`, testCtx.aMigrationStartedEventShouldBeEmitted)
	ctx.Step(`^the event should contain migration metadata$`, testCtx.theEventShouldContainMigrationMetadata)

	ctx.Step(`^a database migration completes successfully$`, testCtx.aDatabaseMigrationCompletesSuccessfully)
	ctx.Step(`^a migration completed event should be emitted$`, testCtx.aMigrationCompletedEventShouldBeEmitted)
	ctx.Step(`^the event should contain migration results$`, testCtx.theEventShouldContainMigrationResults)

	ctx.Step(`^a database migration fails with errors$`, testCtx.aDatabaseMigrationFailsWithErrors)
	ctx.Step(`^a migration failed event should be emitted$`, testCtx.aMigrationFailedEventShouldBeEmitted)
	ctx.Step(`^the event should contain failure details$`, testCtx.theEventShouldContainFailureDetails)
}
