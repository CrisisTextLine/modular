package letsencrypt

import (
	"context"
	"fmt"
	"time"

	"github.com/cucumber/godog"
)

// --- Certificate lifecycle steps ---

func (ctx *LetsEncryptBDDTestContext) aCertificateIsRequestedForDomains() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate certificate request by emitting the appropriate event
	// This tests the event system without requiring actual ACME protocol interaction
	ctx.module.emitEvent(context.Background(), EventTypeCertificateRequested, map[string]interface{}{
		"domains": ctx.config.Domains,
		"count":   len(ctx.config.Domains),
	})

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (ctx *LetsEncryptBDDTestContext) theCertificateIsSuccessfullyIssued() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate successful certificate issuance for each domain
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeCertificateIssued, map[string]interface{}{
			"domain": domain,
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

// --- Certificate renewal steps ---

func (ctx *LetsEncryptBDDTestContext) iHaveExistingCertificatesThatNeedRenewal() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// This step sets up the scenario but doesn't emit events
	// We're simulating having certificates that need renewal
	return nil
}

func (ctx *LetsEncryptBDDTestContext) certificatesAreRenewed() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate certificate renewal for each domain
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeCertificateRenewed, map[string]interface{}{
			"domain": domain,
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (ctx *LetsEncryptBDDTestContext) thereShouldBeARenewalEventForEachDomain() error {
	if ctx.eventObserver == nil || ctx.config == nil {
		return fmt.Errorf("test context not properly initialized")
	}
	expected := make(map[string]bool, len(ctx.config.Domains))
	for _, d := range ctx.config.Domains {
		expected[d] = false
	}
	for _, e := range ctx.eventObserver.GetEvents() {
		if e.Type() == EventTypeCertificateRenewed {
			data := make(map[string]interface{})
			if err := e.DataAs(&data); err == nil {
				if dom, ok := data["domain"].(string); ok {
					if _, present := expected[dom]; present {
						expected[dom] = true
					}
				}
			}
		}
	}
	missing := []string{}
	for d, seen := range expected {
		if !seen {
			missing = append(missing, d)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing renewal events for domains: %v", missing)
	}
	return nil
}

// --- Certificate expiry monitoring steps ---

func (ctx *LetsEncryptBDDTestContext) iHaveCertificatesApproachingExpiry() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// This step sets up the scenario but doesn't emit events
	// We're simulating having certificates approaching expiry
	return nil
}

func (ctx *LetsEncryptBDDTestContext) certificateExpiryMonitoringRuns() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate expiry monitoring for each domain
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeCertificateExpiring, map[string]interface{}{
			"domain":      domain,
			"days_left":   15,
			"expiry_date": time.Now().Add(15 * 24 * time.Hour).Format(time.RFC3339),
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

func (ctx *LetsEncryptBDDTestContext) certificatesHaveExpired() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate expired certificates for each domain
	for _, domain := range ctx.config.Domains {
		ctx.module.emitEvent(context.Background(), EventTypeCertificateExpired, map[string]interface{}{
			"domain":     domain,
			"expired_on": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		})
	}

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

// --- Certificate revocation steps ---

func (ctx *LetsEncryptBDDTestContext) aCertificateIsRevoked() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}

	// Simulate certificate revocation
	ctx.module.emitEvent(context.Background(), EventTypeCertificateRevoked, map[string]interface{}{
		"domain":     ctx.config.Domains[0],
		"reason":     "key_compromise",
		"revoked_on": time.Now().Format(time.RFC3339),
	})

	// Give a small delay to allow event propagation
	time.Sleep(10 * time.Millisecond)

	return nil
}

// --- Certificate failure steps ---

func (ctx *LetsEncryptBDDTestContext) aCertificateRequestFails() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}
	ctx.module.emitEvent(context.Background(), EventTypeError, map[string]interface{}{
		"error":  "order_failed",
		"domain": ctx.config.Domains[0],
		"reason": "acme_server_temporary_error",
	})
	time.Sleep(10 * time.Millisecond)
	return nil
}

// --- Rate limiting steps ---

func (ctx *LetsEncryptBDDTestContext) certificateIssuanceHitsRateLimits() error {
	if ctx.module == nil {
		return fmt.Errorf("module not initialized")
	}
	ctx.module.emitEvent(context.Background(), EventTypeWarning, map[string]interface{}{
		"warning":  "rate_limit_reached",
		"type":     "certificates_per_registered_domain",
		"retry_in": "3600s",
	})
	time.Sleep(10 * time.Millisecond)
	return nil
}

// initCertificateLifecycleBDDSteps registers the certificate lifecycle BDD steps
func initCertificateLifecycleBDDSteps(s *godog.ScenarioContext, ctx *LetsEncryptBDDTestContext) {

	// Certificate lifecycle events
	s.When(`^a certificate is requested for domains$`, ctx.aCertificateIsRequestedForDomains)
	s.When(`^the certificate is successfully issued$`, ctx.theCertificateIsSuccessfullyIssued)

	// Certificate renewal events
	s.Given(`^I have existing certificates that need renewal$`, ctx.iHaveExistingCertificatesThatNeedRenewal)
	s.When(`^certificates are renewed$`, ctx.certificatesAreRenewed)
	s.Then(`^there should be a renewal event for each domain$`, ctx.thereShouldBeARenewalEventForEachDomain)

	// Certificate expiry events (use Step to allow Given/When/Then/And keyword flexibility in aggregated scenario)
	s.Step(`^I have certificates approaching expiry$`, ctx.iHaveCertificatesApproachingExpiry)
	s.When(`^certificate expiry monitoring runs$`, ctx.certificateExpiryMonitoringRuns)
	s.When(`^certificates have expired$`, ctx.certificatesHaveExpired)

	// Certificate revocation events
	s.When(`^a certificate is revoked$`, ctx.aCertificateIsRevoked)

	// Certificate failure path
	s.When(`^a certificate request fails$`, ctx.aCertificateRequestFails)

	// Rate limiting
	s.When(`^certificate issuance hits rate limits$`, ctx.certificateIssuanceHitsRateLimits)
}
