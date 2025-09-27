package letsencrypt

import (
	"testing"

	"github.com/cucumber/godog"
)

// initLetsEncryptBDDSteps centralizes step registration for reuse/clarity
func initLetsEncryptBDDSteps(s *godog.ScenarioContext) {
	// Create a single shared context for all steps
	ctx := &LetsEncryptBDDTestContext{}

	// Register all BDD step groups with the shared context
	initCoreFunctionalityBDDSteps(s, ctx)
	initChallengeTypesBDDSteps(s, ctx)
	initCertificateLifecycleBDDSteps(s, ctx)
	initACMEProtocolBDDSteps(s, ctx)
	initEventSystemBDDSteps(s, ctx)
}

// TestLetsEncryptModuleBDD runs the BDD tests for the LetsEncrypt module
func TestLetsEncryptModuleBDD(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initLetsEncryptBDDSteps,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features/letsencrypt_module.feature"},
			TestingT: t,
			Strict:   true,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
