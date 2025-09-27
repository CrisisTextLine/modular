package letsencrypt

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cucumber/godog"
)

// --- ACME protocol steps ---

func (ctx *LetsEncryptBDDTestContext) aCMEChallengesAreProcessed() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate ACME challenge processing
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeAcmeChallenge, map[string]interface{}{
			"domain":          domain,
			"challenge_type":  "http-01",
			"challenge_token": "test-token-12345",
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (ctx *LetsEncryptBDDTestContext) aCMEAuthorizationIsCompleted() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate ACME authorization completion
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeAcmeAuthorization, map[string]interface{}{
			"domain": domain,
			"status": "valid",
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (ctx *LetsEncryptBDDTestContext) aCMEOrdersAreProcessed() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate ACME order processing
	ctx.module.emitEvent(context.Background(), EventTypeAcmeOrder, map[string]interface{}{
		"domains":  ctx.config.Domains,
		"status":   "ready",
		"order_id": "test-order-12345",
	})

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

// --- Storage operations steps ---

func (ctx *LetsEncryptBDDTestContext) certificatesAreStoredToDisk() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate certificate storage operations
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeStorageWrite, map[string]interface{}{
			"domain": domain,
			"path":   filepath.Join(ctx.config.StoragePath, domain+".crt"),
			"type":   "certificate",
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (ctx *LetsEncryptBDDTestContext) certificatesAreReadFromStorage() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate certificate reading operations
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeStorageRead, map[string]interface{}{
			"domain": domain,
			"path":   filepath.Join(ctx.config.StoragePath, domain+".crt"),
			"type":   "certificate",
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (ctx *LetsEncryptBDDTestContext) storageErrorsOccur() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate storage error
	ctx.module.emitEvent(context.Background(), EventTypeStorageError, map[string]interface{}{
		"error":  "failed to write certificate file",
		"path":   filepath.Join(ctx.config.StoragePath, "test.crt"),
		"domain": "example.com",
	})

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

// --- Configuration operations steps ---

func (ctx *LetsEncryptBDDTestContext) theModuleConfigurationIsLoaded() error {
	// Emit configuration loaded event
	if ctx.module != nil {
		ctx.module.emitEvent(context.Background(), EventTypeConfigLoaded, map[string]interface{}{
			"email":         ctx.config.Email,
			"domains_count": len(ctx.config.Domains),
			"use_staging":   ctx.config.UseStaging,
			"auto_renew":    ctx.config.AutoRenew,
			"dns_enabled":   ctx.config.UseDNS,
		})

		// Give a small delay to allow event propagation
		time.Sleep(10 * time.Millisecond)
	}

	// Continue with the initialization
	return ctx.theLetsEncryptModuleIsInitialized()
}

func (ctx *LetsEncryptBDDTestContext) theConfigurationIsValidated() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate configuration validation
	ctx.module.emitEvent(context.Background(), EventTypeConfigValidated, map[string]interface{}{
		"email":         ctx.config.Email,
		"domains_count": len(ctx.config.Domains),
		"valid":         true,
	})

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

// initACMEProtocolBDDSteps registers the ACME protocol BDD steps
func initACMEProtocolBDDSteps(s *godog.ScenarioContext, ctx *LetsEncryptBDDTestContext) {

	// ACME protocol events
	s.When(`^ACME challenges are processed$`, ctx.aCMEChallengesAreProcessed)
	s.When(`^ACME authorization is completed$`, ctx.aCMEAuthorizationIsCompleted)
	s.When(`^ACME orders are processed$`, ctx.aCMEOrdersAreProcessed)

	// Storage events
	s.When(`^certificates are stored to disk$`, ctx.certificatesAreStoredToDisk)
	s.When(`^certificates are read from storage$`, ctx.certificatesAreReadFromStorage)
	s.When(`^storage errors occur$`, ctx.storageErrorsOccur)

	// Configuration events
	s.When(`^the module configuration is loaded$`, ctx.theModuleConfigurationIsLoaded)
	s.When(`^the configuration is validated$`, ctx.theConfigurationIsValidated)
}
