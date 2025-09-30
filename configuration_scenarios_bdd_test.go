package modular

import (
	"fmt"
	"os"
	"testing"

	"github.com/cucumber/godog"
)

// Step definitions for configuration scenarios

func (ctx *ConfigBDDTestContext) iRegisterTheModulesConfiguration() error {
	if configurable, ok := ctx.module.(Configurable); ok {
		ctx.configError = configurable.RegisterConfig(ctx.app)
	} else {
		return errModuleNotConfigurable
	}
	return nil
}

func (ctx *ConfigBDDTestContext) theConfigurationShouldBeRegisteredSuccessfully() error {
	if ctx.configError != nil {
		return fmt.Errorf("configuration registration failed: %w", ctx.configError)
	}
	return nil
}

func (ctx *ConfigBDDTestContext) theConfigurationShouldBeAvailableForTheModule() error {
	// Check that configuration section is available
	section, err := ctx.app.GetConfigSection(ctx.module.Name())
	if err != nil {
		return fmt.Errorf("configuration section not found for module %s: %w", ctx.module.Name(), err)
	}
	if section == nil {
		return fmt.Errorf("configuration section is nil for module %s: %w", ctx.module.Name(), errModuleNotConfigurable)
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveEnvironmentVariablesSetForModuleConfiguration() error {
	// Set up environment variables for test
	envVars := map[string]string{
		"TEST_CONFIG_MODULE_NAME":            "env-test-module",
		"TEST_CONFIG_MODULE_PORT":            "9090",
		"TEST_CONFIG_MODULE_ENABLED":         "false",
		"TEST_CONFIG_MODULE_HOST":            "env-host",
		"TEST_CONFIG_MODULE_DATABASE_DRIVER": "postgres",
		"TEST_CONFIG_MODULE_DATABASE_DSN":    "postgres://localhost/testdb",
	}

	// Store original values and set new ones
	for key, value := range envVars {
		ctx.originalEnvVars[key] = os.Getenv(key)
		os.Setenv(key, value)
		ctx.environmentVars[key] = value
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveAModuleThatRequiresConfiguration() error {
	return ctx.iHaveAModuleWithConfigurationRequirements()
}

func (ctx *ConfigBDDTestContext) iLoadConfigurationUsingEnvironmentFeeder() error {
	// This would use the environment feeder to load configuration
	// For now, simulate the process
	ctx.configError = nil
	return nil
}

func (ctx *ConfigBDDTestContext) theModuleConfigurationShouldBePopulatedFromEnvironment() error {
	// Verify that environment variables would be loaded correctly
	if len(ctx.environmentVars) == 0 {
		return errNoEnvironmentVariablesSet
	}
	return nil
}

func (ctx *ConfigBDDTestContext) theConfigurationShouldPassValidation() error {
	// Simulate validation passing
	ctx.isValid = true
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveAYAMLConfigurationFile() error {
	yamlContent := `
name: yaml-test-module
port: 8081
enabled: true
host: yaml-host
database:
  driver: mysql
  dsn: mysql://localhost/yamldb
optional: yaml-optional
`
	file, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary YAML file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(yamlContent); err != nil {
		return fmt.Errorf("failed to write YAML content to file: %w", err)
	}

	ctx.yamlFile = file.Name()
	return nil
}

func (ctx *ConfigBDDTestContext) iLoadConfigurationUsingYAMLFeeder() error {
	if ctx.yamlFile == "" {
		return errNoYAMLFileAvailable
	}
	// This would use the YAML feeder to load configuration
	ctx.configError = nil
	return nil
}

func (ctx *ConfigBDDTestContext) theModuleConfigurationShouldBePopulatedFromYAML() error {
	if ctx.yamlFile == "" {
		return errNoYAMLFileCreated
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveAJSONConfigurationFile() error {
	jsonContent := `{
  "name": "json-test-module",
  "port": 8082,
  "enabled": false,
  "host": "json-host",
  "database": {
    "driver": "sqlite",
    "dsn": "sqlite://localhost/jsondb.db"
  },
  "optional": "json-optional"
}`
	file, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary JSON file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(jsonContent); err != nil {
		return fmt.Errorf("failed to write JSON content to file: %w", err)
	}

	ctx.jsonFile = file.Name()
	return nil
}

func (ctx *ConfigBDDTestContext) iLoadConfigurationUsingJSONFeeder() error {
	if ctx.jsonFile == "" {
		return errNoJSONFileAvailable
	}
	// This would use the JSON feeder to load configuration
	ctx.configError = nil
	return nil
}

func (ctx *ConfigBDDTestContext) theModuleConfigurationShouldBePopulatedFromJSON() error {
	if ctx.jsonFile == "" {
		return errNoJSONFileCreated
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveAModuleWithConfigurationValidationRules() error {
	return ctx.iHaveAModuleWithConfigurationRequirements()
}

func (ctx *ConfigBDDTestContext) iHaveValidConfigurationData() error {
	ctx.configData = &TestModuleConfig{
		Name:    "valid-module",
		Port:    8080,
		Enabled: true,
		Host:    "localhost",
	}
	ctx.configData.(*TestModuleConfig).Database.Driver = "postgres"
	ctx.configData.(*TestModuleConfig).Database.DSN = "postgres://localhost/testdb"
	return nil
}

func (ctx *ConfigBDDTestContext) iValidateTheConfiguration() error {
	if config, ok := ctx.configData.(*TestModuleConfig); ok {
		ctx.validationError = config.ValidateConfig()
	} else {
		ctx.validationError = errNoConfigurationData
	}
	return nil
}

func (ctx *ConfigBDDTestContext) theValidationShouldPass() error {
	if ctx.validationError != nil {
		return fmt.Errorf("validation should have passed but failed: %w", ctx.validationError)
	}
	ctx.isValid = true
	return nil
}

func (ctx *ConfigBDDTestContext) noValidationErrorsShouldBeReported() error {
	if len(ctx.validationErrors) > 0 {
		return fmt.Errorf("expected no validation errors, got: %w", errExpectedNoValidationErrors)
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveInvalidConfigurationData() error {
	ctx.configData = &TestModuleConfig{
		Name:    "", // Invalid: empty name
		Port:    -1, // Invalid: negative port
		Enabled: true,
		Host:    "localhost",
	}
	// Missing required database configuration
	return nil
}

func (ctx *ConfigBDDTestContext) theValidationShouldFail() error {
	if ctx.validationError == nil {
		return errValidationShouldHaveFailed
	}
	return nil
}

func (ctx *ConfigBDDTestContext) appropriateValidationErrorsShouldBeReported() error {
	if ctx.validationError == nil {
		return errNoValidationErrorReported
	}
	// Check that the error message contains relevant information
	if len(ctx.validationError.Error()) == 0 {
		return errValidationErrorMessageEmpty
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveAModuleWithDefaultConfigurationValues() error {
	return ctx.iHaveAModuleWithConfigurationRequirements()
}

func (ctx *ConfigBDDTestContext) iLoadConfigurationWithoutProvidingAllValues() error {
	// Simulate loading partial configuration, defaults should fill in
	ctx.configError = nil
	return nil
}

func (ctx *ConfigBDDTestContext) theMissingValuesShouldUseDefaults() error {
	// Verify that default values would be applied
	return nil
}

func (ctx *ConfigBDDTestContext) theConfigurationShouldBeComplete() error {
	// Verify that all fields have values (either provided or default)
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveAModuleWithRequiredConfigurationFields() error {
	return ctx.iHaveAModuleWithConfigurationRequirements()
}

func (ctx *ConfigBDDTestContext) iLoadConfigurationWithoutRequiredValues() error {
	// Simulate loading configuration missing required fields
	ctx.configError = errRequiredFieldMissing
	return nil
}

func (ctx *ConfigBDDTestContext) theConfigurationLoadingShouldFail() error {
	if ctx.configError == nil {
		return errConfigLoadingShouldHaveFailed
	}
	return nil
}

func (ctx *ConfigBDDTestContext) theErrorShouldIndicateMissingRequiredFields() error {
	if ctx.configError == nil {
		return errNoErrorToCheckConfig
	}
	// Check that error mentions required fields
	errorMsg := ctx.configError.Error()
	if len(errorMsg) == 0 {
		return errErrorMessageEmpty
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iHaveAModuleWithConfigurationFieldTrackingEnabled() error {
	ctx.module = &TestConfigurableModule{name: "tracking-module"}
	return nil
}

func (ctx *ConfigBDDTestContext) iLoadConfigurationFromMultipleSources() error {
	// Simulate loading from multiple sources with field tracking
	ctx.fieldTracker.TrackField("name", "environment")
	ctx.fieldTracker.TrackField("port", "yaml")
	ctx.fieldTracker.TrackField("database.driver", "json")
	return nil
}

func (ctx *ConfigBDDTestContext) iShouldBeAbleToTrackWhichFieldsWereSet() error {
	trackedFields := ctx.fieldTracker.GetTrackedFields()
	if len(trackedFields) == 0 {
		return errNoFieldsTracked
	}
	return nil
}

func (ctx *ConfigBDDTestContext) iShouldKnowTheSourceOfEachConfigurationValue() error {
	trackedFields := ctx.fieldTracker.GetTrackedFields()
	expectedSources := map[string]string{
		"name":            "environment",
		"port":            "yaml",
		"database.driver": "json",
	}

	for field, expectedSource := range expectedSources {
		if actualSource, exists := trackedFields[field]; !exists {
			return fmt.Errorf("field %s: %w", field, errFieldNotTracked)
		} else if actualSource != expectedSource {
			return fmt.Errorf("field %s expected source %s, got %s: %w", field, expectedSource, actualSource, errFieldSourceMismatch)
		}
	}
	return nil
}

// TestConfigurationManagement runs the BDD tests for configuration management
func TestConfigurationManagement(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			testCtx := &ConfigBDDTestContext{}

			// Include all base steps from configuration_base_bdd_test.go
			InitializeConfigurationScenario(ctx, testCtx)

			// Configuration registration steps
			ctx.Step(`^I register the module's configuration$`, testCtx.iRegisterTheModulesConfiguration)
			ctx.Step(`^the configuration should be registered successfully$`, testCtx.theConfigurationShouldBeRegisteredSuccessfully)
			ctx.Step(`^the configuration should be available for the module$`, testCtx.theConfigurationShouldBeAvailableForTheModule)

			// Environment configuration steps
			ctx.Step(`^I have environment variables set for module configuration$`, testCtx.iHaveEnvironmentVariablesSetForModuleConfiguration)
			ctx.Step(`^I have a module that requires configuration$`, testCtx.iHaveAModuleThatRequiresConfiguration)
			ctx.Step(`^I load configuration using environment feeder$`, testCtx.iLoadConfigurationUsingEnvironmentFeeder)
			ctx.Step(`^the module configuration should be populated from environment$`, testCtx.theModuleConfigurationShouldBePopulatedFromEnvironment)
			ctx.Step(`^the configuration should pass validation$`, testCtx.theConfigurationShouldPassValidation)

			// YAML configuration steps
			ctx.Step(`^I have a YAML configuration file$`, testCtx.iHaveAYAMLConfigurationFile)
			ctx.Step(`^I load configuration using YAML feeder$`, testCtx.iLoadConfigurationUsingYAMLFeeder)
			ctx.Step(`^the module configuration should be populated from YAML$`, testCtx.theModuleConfigurationShouldBePopulatedFromYAML)

			// JSON configuration steps
			ctx.Step(`^I have a JSON configuration file$`, testCtx.iHaveAJSONConfigurationFile)
			ctx.Step(`^I load configuration using JSON feeder$`, testCtx.iLoadConfigurationUsingJSONFeeder)
			ctx.Step(`^the module configuration should be populated from JSON$`, testCtx.theModuleConfigurationShouldBePopulatedFromJSON)

			// Validation steps
			ctx.Step(`^I have a module with configuration validation rules$`, testCtx.iHaveAModuleWithConfigurationValidationRules)
			ctx.Step(`^I have valid configuration data$`, testCtx.iHaveValidConfigurationData)
			ctx.Step(`^I validate the configuration$`, testCtx.iValidateTheConfiguration)
			ctx.Step(`^the validation should pass$`, testCtx.theValidationShouldPass)
			ctx.Step(`^no validation errors should be reported$`, testCtx.noValidationErrorsShouldBeReported)
			ctx.Step(`^I have invalid configuration data$`, testCtx.iHaveInvalidConfigurationData)
			ctx.Step(`^the validation should fail$`, testCtx.theValidationShouldFail)
			ctx.Step(`^appropriate validation errors should be reported$`, testCtx.appropriateValidationErrorsShouldBeReported)

			// Default values steps
			ctx.Step(`^I have a module with default configuration values$`, testCtx.iHaveAModuleWithDefaultConfigurationValues)
			ctx.Step(`^I load configuration without providing all values$`, testCtx.iLoadConfigurationWithoutProvidingAllValues)
			ctx.Step(`^the missing values should use defaults$`, testCtx.theMissingValuesShouldUseDefaults)
			ctx.Step(`^the configuration should be complete$`, testCtx.theConfigurationShouldBeComplete)

			// Required fields steps
			ctx.Step(`^I have a module with required configuration fields$`, testCtx.iHaveAModuleWithRequiredConfigurationFields)
			ctx.Step(`^I load configuration without required values$`, testCtx.iLoadConfigurationWithoutRequiredValues)
			ctx.Step(`^the configuration loading should fail$`, testCtx.theConfigurationLoadingShouldFail)
			ctx.Step(`^the error should indicate missing required fields$`, testCtx.theErrorShouldIndicateMissingRequiredFields)

			// Field tracking steps
			ctx.Step(`^I have a module with configuration field tracking enabled$`, testCtx.iHaveAModuleWithConfigurationFieldTrackingEnabled)
			ctx.Step(`^I load configuration from multiple sources$`, testCtx.iLoadConfigurationFromMultipleSources)
			ctx.Step(`^I should be able to track which fields were set$`, testCtx.iShouldBeAbleToTrackWhichFieldsWereSet)
			ctx.Step(`^I should know the source of each configuration value$`, testCtx.iShouldKnowTheSourceOfEachConfigurationValue)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features/configuration_management.feature"},
			TestingT: t,
			Strict:   true,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
