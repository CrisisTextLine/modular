package reverseproxy

import (
	"testing"

	"github.com/cucumber/godog"
)

// TestReverseProxyModuleBDD runs the BDD tests for the ReverseProxy module
// This test aggregates scenarios from all the split BDD test files
func TestReverseProxyModuleBDD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping BDD tests in short mode")
	}
	suite := godog.TestSuite{
		ScenarioInitializer: func(s *godog.ScenarioContext) {
			ctx := &ReverseProxyBDDTestContext{}

			// Register all step definitions from all BDD files
			registerAllStepDefinitions(s, ctx)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
			Strict:   true, // fail suite on undefined or pending steps
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
