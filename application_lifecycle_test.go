package modular

import (
	"log/slog"
	"testing"
)

// Test_ApplicationLifecycle tests the Start and Stop methods
func Test_ApplicationLifecycle(t *testing.T) {
	// Test successful Start and Stop
	t.Run("Successful lifecycle", func(t *testing.T) {
		app := &StdApplication{
			cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
			cfgSections:    make(map[string]ConfigProvider),
			svcRegistry:    make(ServiceRegistry),
			moduleRegistry: make(ModuleRegistry),
			logger:         &logger{t},
		}

		module1 := &lifecycleTestModule{testModule: testModule{name: "module1"}}
		module2 := &lifecycleTestModule{testModule: testModule{name: "module2", dependencies: []string{"module1"}}}

		app.RegisterModule(module1)
		app.RegisterModule(module2)

		// Test Start
		if err := app.Start(); err != nil {
			t.Errorf("Start() error = %v, expected no error", err)
		}

		// Verify context was created
		if app.ctx == nil {
			t.Error("Start() did not create application context")
		}

		// Verify modules were started in correct order
		if !module1.startCalled {
			t.Error("Start() did not call Start on first module")
		}
		if !module2.startCalled {
			t.Error("Start() did not call Start on second module")
		}

		// Test Stop
		if err := app.Stop(); err != nil {
			t.Errorf("Stop() error = %v, expected no error", err)
		}

		// Verify modules were stopped (should be in reverse order)
		if !module1.stopCalled {
			t.Error("Stop() did not call Stop on first module")
		}
		if !module2.stopCalled {
			t.Error("Stop() did not call Stop on second module")
		}
	})

	// Test Start failure
	t.Run("Start failure", func(t *testing.T) {
		app := &StdApplication{
			cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
			cfgSections:    make(map[string]ConfigProvider),
			svcRegistry:    make(ServiceRegistry),
			moduleRegistry: make(ModuleRegistry),
			logger:         &logger{t},
		}

		failingModule := &lifecycleTestModule{
			testModule: testModule{name: "failing"},
			startError: ErrModuleStartFailed,
		}

		app.RegisterModule(failingModule)

		// Test Start
		if err := app.Start(); err == nil {
			t.Error("Start() expected error for failing module, got nil")
		}
	})

	// Test Stop with error
	t.Run("Stop with error", func(t *testing.T) {
		app := &StdApplication{
			cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
			cfgSections:    make(map[string]ConfigProvider),
			svcRegistry:    make(ServiceRegistry),
			moduleRegistry: make(ModuleRegistry),
			logger:         slog.Default(),
		}

		failingModule := &lifecycleTestModule{
			testModule: testModule{name: "failing"},
			stopError:  ErrModuleStopFailed,
		}

		app.RegisterModule(failingModule)

		// Start first so we can test Stop
		if err := app.Start(); err != nil {
			t.Fatalf("Start() error = %v, expected no error", err)
		}

		// Test Stop - should return error but continue stopping
		if err := app.Stop(); err == nil {
			t.Error("Stop() expected error for failing module, got nil")
		}
	})
}
