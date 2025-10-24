package modular

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
)

// Test_OnConfigLoaded_SingleHook tests basic hook registration and execution
func Test_OnConfigLoaded_SingleHook(t *testing.T) {
	logger := &MockLogger{}
	// Allow any Debug calls
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	config := &struct {
		Value string `yaml:"value" default:"default"`
	}{}
	
	app := NewStdApplication(NewStdConfigProvider(config), logger)
	
	hookExecuted := false
	app.OnConfigLoaded(func(app Application) error {
		hookExecuted = true
		return nil
	})
	
	// Add a dummy module so we exercise full init path
	testModule := &TestModuleWithCachedLogger{}
	app.RegisterModule(testModule)
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	if !hookExecuted {
		t.Error("Hook was not executed during Init")
	}
}

// Test_OnConfigLoaded_MultipleHooks tests that multiple hooks execute in order
func Test_OnConfigLoaded_MultipleHooks(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	config := &struct {
		Value string `yaml:"value" default:"default"`
	}{}
	
	app := NewStdApplication(NewStdConfigProvider(config), logger)
	
	executionOrder := []int{}
	
	app.OnConfigLoaded(func(app Application) error {
		executionOrder = append(executionOrder, 1)
		return nil
	})
	
	app.OnConfigLoaded(func(app Application) error {
		executionOrder = append(executionOrder, 2)
		return nil
	})
	
	app.OnConfigLoaded(func(app Application) error {
		executionOrder = append(executionOrder, 3)
		return nil
	})
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 hooks to execute, got %d", len(executionOrder))
	}
	
	for i, order := range executionOrder {
		if order != i+1 {
			t.Errorf("Hook %d executed out of order, got position %d", i+1, order)
		}
	}
}

// Test_OnConfigLoaded_HookError tests that hook errors are properly propagated
func Test_OnConfigLoaded_HookError(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	config := &struct {
		Value string `yaml:"value" default:"default"`
	}{}
	
	app := NewStdApplication(NewStdConfigProvider(config), logger)
	
	expectedError := fmt.Errorf("hook error")
	app.OnConfigLoaded(func(app Application) error {
		return expectedError
	})
	
	err := app.Init()
	if err == nil {
		t.Fatal("Expected Init to fail when hook returns error")
	}
	
	if !containsSubstring(err.Error(), "config loaded hook") {
		t.Errorf("Expected error message to mention 'config loaded hook', got: %v", err)
	}
}

// Test_OnConfigLoaded_LoggerReconfiguration tests the main use case: reconfiguring logger based on config
func Test_OnConfigLoaded_LoggerReconfiguration(t *testing.T) {
	initialLogger := &MockLogger{}
	initialLogger.On("Debug", mock.Anything, mock.Anything).Return()
	initialLogger.On("Info", mock.Anything, mock.Anything).Return()
	
	type AppConfig struct {
		LogLevel string `yaml:"logLevel" default:"info"`
	}
	
	config := &AppConfig{}
	app := NewStdApplication(NewStdConfigProvider(config), initialLogger)
	
	// Create a test module that caches the logger
	testModule := &TestModuleWithCachedLogger{}
	app.RegisterModule(testModule)
	
	// Register hook to reconfigure logger based on config
	newLogger := &MockLogger{}
	newLogger.On("Debug", mock.Anything, mock.Anything).Return()
	newLogger.On("Info", mock.Anything, mock.Anything).Return()
	
	app.OnConfigLoaded(func(app Application) error {
		cfg := app.ConfigProvider().GetConfig().(*AppConfig)
		if cfg.LogLevel == "debug" {
			app.SetLogger(newLogger)
		}
		return nil
	})
	
	// Set config to trigger logger change
	config.LogLevel = "debug"
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	// Verify the module received the new logger, not the initial one
	if testModule.logger == initialLogger {
		t.Error("Module still has initial logger, expected new logger")
	}
	
	if testModule.logger != newLogger {
		t.Error("Module does not have the reconfigured logger")
	}
	
	// Verify app.Logger() returns the new logger
	if app.Logger() != newLogger {
		t.Error("Application Logger() does not return reconfigured logger")
	}
}

// Test_OnConfigLoaded_AccessConfig tests that hooks can access loaded config
func Test_OnConfigLoaded_AccessConfig(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	type AppConfig struct {
		Setting string `yaml:"setting" default:"test_value"`
	}
	
	config := &AppConfig{}
	app := NewStdApplication(NewStdConfigProvider(config), logger)
	
	var capturedSetting string
	app.OnConfigLoaded(func(app Application) error {
		cfg := app.ConfigProvider().GetConfig().(*AppConfig)
		capturedSetting = cfg.Setting
		return nil
	})
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	if capturedSetting != "test_value" {
		t.Errorf("Hook did not access config correctly, got: %s", capturedSetting)
	}
}

// Test_OnConfigLoaded_WithBuilder tests hook registration via builder
func Test_OnConfigLoaded_WithBuilder(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	type AppConfig struct {
		Value string `yaml:"value" default:"test"`
	}
	
	config := &AppConfig{}
	
	hookExecuted := false
	app, err := NewApplication(
		WithLogger(logger),
		WithConfigProvider(NewStdConfigProvider(config)),
		WithOnConfigLoaded(func(app Application) error {
			hookExecuted = true
			return nil
		}),
	)
	
	if err != nil {
		t.Fatalf("NewApplication failed: %v", err)
	}
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	if !hookExecuted {
		t.Error("Hook registered via builder was not executed")
	}
}

// Test_OnConfigLoaded_MultipleHooksViaBuilder tests multiple hooks via builder
func Test_OnConfigLoaded_MultipleHooksViaBuilder(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	type AppConfig struct {
		Value string `yaml:"value" default:"test"`
	}
	
	config := &AppConfig{}
	
	executionCount := 0
	
	app, err := NewApplication(
		WithLogger(logger),
		WithConfigProvider(NewStdConfigProvider(config)),
		WithOnConfigLoaded(
			func(app Application) error {
				executionCount++
				return nil
			},
			func(app Application) error {
				executionCount++
				return nil
			},
		),
	)
	
	if err != nil {
		t.Fatalf("NewApplication failed: %v", err)
	}
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	if executionCount != 2 {
		t.Errorf("Expected 2 hooks to execute, got %d", executionCount)
	}
}

// Test_OnConfigLoaded_NilHook tests that nil hooks are ignored
func Test_OnConfigLoaded_NilHook(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	config := &struct {
		Value string `yaml:"value" default:"default"`
	}{}
	
	app := NewStdApplication(NewStdConfigProvider(config), logger)
	
	// Register nil hook - should be ignored
	app.OnConfigLoaded(nil)
	
	// Should not panic and init should succeed
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

// Test_OnConfigLoaded_ExecutesBeforeModuleInit tests timing of hook execution
func Test_OnConfigLoaded_ExecutesBeforeModuleInit(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	type AppConfig struct {
		Value string `yaml:"value" default:"initial"`
	}
	
	config := &AppConfig{}
	app := NewStdApplication(NewStdConfigProvider(config), logger)
	
	// Track execution order
	var executionOrder []string
	
	// Module that records when it initializes
	module := &ConfigLoadedTestModule{
		name: "test",
		initFunc: func(app Application) error {
			executionOrder = append(executionOrder, "module_init")
			return nil
		},
	}
	app.RegisterModule(module)
	
	// Hook that records when it executes
	app.OnConfigLoaded(func(app Application) error {
		executionOrder = append(executionOrder, "hook")
		return nil
	})
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	// Verify hook executed before module init
	if len(executionOrder) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(executionOrder))
	}
	
	if executionOrder[0] != "hook" {
		t.Errorf("Expected hook to execute first, got: %s", executionOrder[0])
	}
	
	if executionOrder[1] != "module_init" {
		t.Errorf("Expected module_init to execute second, got: %s", executionOrder[1])
	}
}

// Test_OnConfigLoaded_CanModifyServices tests that hooks can register services
func Test_OnConfigLoaded_CanModifyServices(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	logger.On("Info", mock.Anything, mock.Anything).Return()
	
	config := &struct{}{}
	
	app := NewStdApplication(NewStdConfigProvider(config), logger)
	
	// Hook that registers a service
	app.OnConfigLoaded(func(app Application) error {
		return app.RegisterService("test_service", "test_value")
	})
	
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	
	// Verify service was registered
	var service string
	if err := app.GetService("test_service", &service); err != nil {
		t.Errorf("Service not found: %v", err)
	}
	
	if service != "test_value" {
		t.Errorf("Expected service value 'test_value', got: %s", service)
	}
}

// TestModuleWithCachedLogger is a test module that caches the logger reference
type TestModuleWithCachedLogger struct {
	logger Logger
}

func (m *TestModuleWithCachedLogger) Name() string {
	return "test_module_with_logger"
}

func (m *TestModuleWithCachedLogger) Init(app Application) error {
	m.logger = app.Logger()
	return nil
}

// ConfigLoadedTestModule is a test module with configurable init behavior
type ConfigLoadedTestModule struct {
	name     string
	initFunc func(app Application) error
}

func (m *ConfigLoadedTestModule) Name() string {
	return m.name
}

func (m *ConfigLoadedTestModule) Init(app Application) error {
	if m.initFunc != nil {
		return m.initFunc(app)
	}
	return nil
}

// Helper function
func containsSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

