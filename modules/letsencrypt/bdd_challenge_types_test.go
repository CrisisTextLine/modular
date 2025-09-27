package letsencrypt

import (
	"fmt"

	"github.com/cucumber/godog"
)

// --- HTTP-01 challenge configuration ---

func (ctx *LetsEncryptBDDTestContext) iHaveLetsEncryptConfiguredForHTTP01Challenge() error {
	err := ctx.iHaveAModularApplicationWithLetsEncryptModuleConfigured()
	if err != nil {
		return err
	}

	// Configure for HTTP-01 challenge
	ctx.config.UseDNS = false
	ctx.config.HTTPProvider = &HTTPProviderConfig{
		UseBuiltIn: true,
		Port:       8080,
	}

	// Recreate module with updated config
	ctx.module, err = New(ctx.config)
	if err != nil {
		ctx.lastError = err
		return err
	}

	return nil
}

func (ctx *LetsEncryptBDDTestContext) theModuleIsInitializedWithHTTPChallengeType() error {
	return ctx.theLetsEncryptModuleIsInitialized()
}

func (ctx *LetsEncryptBDDTestContext) theHTTPChallengeHandlerShouldBeConfigured() error {
	if ctx.module == nil || ctx.module.config.HTTPProvider == nil {
		return fmt.Errorf("HTTP challenge handler not configured")
	}

	if !ctx.module.config.HTTPProvider.UseBuiltIn {
		return fmt.Errorf("built-in HTTP provider not enabled")
	}

	return nil
}

func (ctx *LetsEncryptBDDTestContext) theModuleShouldBeReadyForDomainValidation() error {
	// Verify HTTP challenge configuration
	if ctx.module.config.UseDNS {
		return fmt.Errorf("DNS mode enabled when HTTP mode expected")
	}
	return nil
}

// --- DNS-01 challenge configuration ---

func (ctx *LetsEncryptBDDTestContext) iHaveLetsEncryptConfiguredForDNS01ChallengeWithCloudflare() error {
	err := ctx.iHaveAModularApplicationWithLetsEncryptModuleConfigured()
	if err != nil {
		return err
	}

	// Configure for DNS-01 challenge with Cloudflare (clear HTTP provider first)
	ctx.config.UseDNS = true
	ctx.config.HTTPProvider = nil // Clear HTTP provider to avoid conflict
	ctx.config.DNSProvider = &DNSProviderConfig{
		Provider: "cloudflare",
		Cloudflare: &CloudflareConfig{
			Email:    "test@example.com",
			APIToken: "test-token",
		},
	}

	// Recreate module with updated config
	ctx.module, err = New(ctx.config)
	if err != nil {
		ctx.lastError = err
		return err
	}

	return nil
}

func (ctx *LetsEncryptBDDTestContext) theModuleIsInitializedWithDNSChallengeType() error {
	return ctx.theLetsEncryptModuleIsInitialized()
}

func (ctx *LetsEncryptBDDTestContext) theDNSChallengeHandlerShouldBeConfigured() error {
	if ctx.module == nil || ctx.module.config.DNSProvider == nil {
		return fmt.Errorf("DNS challenge handler not configured")
	}

	if ctx.module.config.DNSProvider.Provider != "cloudflare" {
		return fmt.Errorf("expected cloudflare provider, got %s", ctx.module.config.DNSProvider.Provider)
	}

	return nil
}

func (ctx *LetsEncryptBDDTestContext) theModuleShouldBeReadyForDNSValidation() error {
	// Verify DNS challenge configuration
	if !ctx.module.config.UseDNS {
		return fmt.Errorf("DNS mode not enabled")
	}
	return nil
}

// --- Mixed challenge reconfiguration ---

func (ctx *LetsEncryptBDDTestContext) reconfigureToDNS01ChallengeWithCloudflare() error {
	if ctx.config == nil {
		return fmt.Errorf("no existing config to modify")
	}
	ctx.config.UseDNS = true
	ctx.config.HTTPProvider = nil
	ctx.config.DNSProvider = &DNSProviderConfig{
		Provider: "cloudflare",
		Cloudflare: &CloudflareConfig{
			Email:    "test@example.com",
			APIToken: "updated-token",
		},
	}
	mod, err := New(ctx.config)
	if err != nil {
		ctx.lastError = err
		return err
	}
	ctx.module = mod
	return nil
}

// initChallengeTypesBDDSteps registers the challenge types BDD steps
func initChallengeTypesBDDSteps(s *godog.ScenarioContext, ctx *LetsEncryptBDDTestContext) {

	// HTTP-01 challenge
	s.Given(`^I have LetsEncrypt configured for HTTP-01 challenge$`, ctx.iHaveLetsEncryptConfiguredForHTTP01Challenge)
	s.When(`^the module is initialized with HTTP challenge type$`, ctx.theModuleIsInitializedWithHTTPChallengeType)
	s.Then(`^the HTTP challenge handler should be configured$`, ctx.theHTTPChallengeHandlerShouldBeConfigured)
	s.Then(`^the module should be ready for domain validation$`, ctx.theModuleShouldBeReadyForDomainValidation)

	// DNS-01 challenge
	s.Given(`^I have LetsEncrypt configured for DNS-01 challenge with Cloudflare$`, ctx.iHaveLetsEncryptConfiguredForDNS01ChallengeWithCloudflare)
	s.When(`^the module is initialized with DNS challenge type$`, ctx.theModuleIsInitializedWithDNSChallengeType)
	s.Then(`^the DNS challenge handler should be configured$`, ctx.theDNSChallengeHandlerShouldBeConfigured)
	s.Then(`^the module should be ready for DNS validation$`, ctx.theModuleShouldBeReadyForDNSValidation)

	// Mixed challenge reconfiguration
	s.When(`^I reconfigure to DNS-01 challenge with Cloudflare$`, ctx.reconfigureToDNS01ChallengeWithCloudflare)
}
