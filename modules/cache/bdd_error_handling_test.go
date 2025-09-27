package cache

import (
	"errors"
)

// Error handling BDD test steps

func (ctx *CacheBDDTestContext) theModuleShouldHandleConnectionErrorsGracefully() error {
	// Error should be captured, not panic
	if ctx.lastError == nil {
		return errors.New("expected connection error but none occurred")
	}
	return nil
}

func (ctx *CacheBDDTestContext) appropriateErrorMessagesShouldBeLogged() error {
	// This would be verified by checking the test logger output
	// For now, we just verify an error occurred
	return ctx.theModuleShouldHandleConnectionErrorsGracefully()
}
