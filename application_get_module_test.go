package modular

import (
	"testing"
)

// Test_GetModule tests the GetModule method
func Test_GetModule(t *testing.T) {
	tests := []struct {
		name       string
		modules    []Module
		lookupName string
		wantNil    bool
	}{
		{
			name: "Get existing module",
			modules: []Module{
				&testModule{name: "module-a"},
				&testModule{name: "module-b"},
			},
			lookupName: "module-a",
			wantNil:    false,
		},
		{
			name: "Get non-existent module",
			modules: []Module{
				&testModule{name: "module-a"},
			},
			lookupName: "module-b",
			wantNil:    true,
		},
		{
			name:       "Get from empty registry",
			modules:    []Module{},
			lookupName: "module-a",
			wantNil:    true,
		},
		{
			name: "Get multiple modules by name",
			modules: []Module{
				&testModule{name: "database"},
				&testModule{name: "cache"},
				&testModule{name: "httpserver"},
			},
			lookupName: "cache",
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &StdApplication{
				cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
				cfgSections:    make(map[string]ConfigProvider),
				svcRegistry:    make(ServiceRegistry),
				moduleRegistry: make(ModuleRegistry),
				logger:         &logger{t},
			}

			// Register modules
			for _, module := range tt.modules {
				app.RegisterModule(module)
			}

			// Get module
			result := app.GetModule(tt.lookupName)

			if tt.wantNil {
				if result != nil {
					t.Errorf("GetModule(%s) = %v, want nil", tt.lookupName, result)
				}
			} else {
				if result == nil {
					t.Errorf("GetModule(%s) = nil, want non-nil", tt.lookupName)
				} else if result.Name() != tt.lookupName {
					t.Errorf("GetModule(%s).Name() = %s, want %s", tt.lookupName, result.Name(), tt.lookupName)
				}
			}
		})
	}
}

// Test_GetModule_TypeAssertion tests type assertions with GetModule
func Test_GetModule_TypeAssertion(t *testing.T) {
	app := &StdApplication{
		cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
		cfgSections:    make(map[string]ConfigProvider),
		svcRegistry:    make(ServiceRegistry),
		moduleRegistry: make(ModuleRegistry),
		logger:         &logger{t},
	}

	// Register a specific module type
	configModule := &configRegisteringModule{
		testModule: testModule{name: "config-module"},
	}
	app.RegisterModule(configModule)

	// Test type assertion
	module := app.GetModule("config-module")
	if module == nil {
		t.Fatal("GetModule returned nil")
	}

	// Type assert to specific module type
	specificModule, ok := module.(*configRegisteringModule)
	if !ok {
		t.Errorf("Type assertion to *configRegisteringModule failed")
	}
	if specificModule.Name() != "config-module" {
		t.Errorf("Module name = %s, want config-module", specificModule.Name())
	}
}

// Test_GetAllModules tests the GetAllModules method
func Test_GetAllModules(t *testing.T) {
	tests := []struct {
		name          string
		modules       []Module
		expectedCount int
		checkNames    []string
	}{
		{
			name: "Get all modules",
			modules: []Module{
				&testModule{name: "module-a"},
				&testModule{name: "module-b"},
				&testModule{name: "module-c"},
			},
			expectedCount: 3,
			checkNames:    []string{"module-a", "module-b", "module-c"},
		},
		{
			name:          "Get from empty registry",
			modules:       []Module{},
			expectedCount: 0,
			checkNames:    []string{},
		},
		{
			name: "Get single module",
			modules: []Module{
				&testModule{name: "database"},
			},
			expectedCount: 1,
			checkNames:    []string{"database"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &StdApplication{
				cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
				cfgSections:    make(map[string]ConfigProvider),
				svcRegistry:    make(ServiceRegistry),
				moduleRegistry: make(ModuleRegistry),
				logger:         &logger{t},
			}

			// Register modules
			for _, module := range tt.modules {
				app.RegisterModule(module)
			}

			// Get all modules
			result := app.GetAllModules()

			// Check count
			if len(result) != tt.expectedCount {
				t.Errorf("GetAllModules() returned %d modules, want %d", len(result), tt.expectedCount)
			}

			// Check that all expected names are present
			for _, name := range tt.checkNames {
				module, exists := result[name]
				if !exists {
					t.Errorf("GetAllModules() missing module %s", name)
				}
				if module.Name() != name {
					t.Errorf("Module name = %s, want %s", module.Name(), name)
				}
			}
		})
	}
}

// Test_GetAllModules_ReturnsCopy tests that GetAllModules returns a copy
func Test_GetAllModules_ReturnsCopy(t *testing.T) {
	app := &StdApplication{
		cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
		cfgSections:    make(map[string]ConfigProvider),
		svcRegistry:    make(ServiceRegistry),
		moduleRegistry: make(ModuleRegistry),
		logger:         &logger{t},
	}

	// Register modules
	app.RegisterModule(&testModule{name: "module-a"})
	app.RegisterModule(&testModule{name: "module-b"})

	// Get all modules
	modules1 := app.GetAllModules()
	modules2 := app.GetAllModules()

	// Verify both calls return the expected modules
	if len(modules1) != 2 {
		t.Errorf("First GetAllModules() returned %d modules, want 2", len(modules1))
	}
	if len(modules2) != 2 {
		t.Errorf("Second GetAllModules() returned %d modules, want 2", len(modules2))
	}

	// Modify the first returned map
	modules1["module-c"] = &testModule{name: "module-c"}

	// Verify the second returned map is unchanged
	if len(modules2) != 2 {
		t.Errorf("Second map was modified, len = %d, want 2", len(modules2))
	}
	if _, exists := modules2["module-c"]; exists {
		t.Error("Second map contains module-c, should not be affected by first map modification")
	}

	// Verify internal registry is unchanged
	modules3 := app.GetAllModules()
	if len(modules3) != 2 {
		t.Errorf("Internal registry was modified, GetAllModules() returned %d modules, want 2", len(modules3))
	}
	if _, exists := modules3["module-c"]; exists {
		t.Error("Internal registry contains module-c, should not be affected by external map modification")
	}
}

// Test_GetModule_AfterInit tests that GetModule works after Init
func Test_GetModule_AfterInit(t *testing.T) {
	// Setup standard config and logger for tests
	stdConfig := NewStdConfigProvider(testCfg{Str: "test"})
	stdLogger := &logger{t}

	// Setup mock AppConfigLoader
	originalLoader := AppConfigLoader
	defer func() { AppConfigLoader = originalLoader }()
	AppConfigLoader = testAppConfigLoader

	app := &StdApplication{
		cfgProvider:    stdConfig,
		cfgSections:    make(map[string]ConfigProvider),
		svcRegistry:    make(ServiceRegistry),
		moduleRegistry: make(ModuleRegistry),
		logger:         stdLogger,
	}

	// Register modules
	app.RegisterModule(&testModule{name: "module-a"})
	app.RegisterModule(&testModule{name: "module-b", dependencies: []string{"module-a"}})

	// Initialize the application
	err := app.Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test GetModule after Init
	moduleA := app.GetModule("module-a")
	if moduleA == nil {
		t.Error("GetModule(module-a) returned nil after Init")
	}

	moduleB := app.GetModule("module-b")
	if moduleB == nil {
		t.Error("GetModule(module-b) returned nil after Init")
	}

	// Test GetAllModules after Init
	allModules := app.GetAllModules()
	if len(allModules) != 2 {
		t.Errorf("GetAllModules() returned %d modules after Init, want 2", len(allModules))
	}
}

// Test_GetModule_OptionalDependency demonstrates the optional dependency use case
func Test_GetModule_OptionalDependency(t *testing.T) {
	app := &StdApplication{
		cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
		cfgSections:    make(map[string]ConfigProvider),
		svcRegistry:    make(ServiceRegistry),
		moduleRegistry: make(ModuleRegistry),
		logger:         &logger{t},
	}

	// Register only the main module, not the optional one
	app.RegisterModule(&testModule{name: "main-module"})

	// Check if optional module is available (it's not)
	optionalModule := app.GetModule("optional-module")
	if optionalModule != nil {
		t.Error("Expected optional-module to be nil, but got a module")
	}

	// Now register the optional module
	app.RegisterModule(&testModule{name: "optional-module"})

	// Check again - should now be available
	optionalModule = app.GetModule("optional-module")
	if optionalModule == nil {
		t.Error("Expected optional-module to be available after registration")
	}
}

// Test_GetAllModules_Introspection demonstrates the introspection use case
func Test_GetAllModules_Introspection(t *testing.T) {
	app := &StdApplication{
		cfgProvider:    NewStdConfigProvider(testCfg{Str: "test"}),
		cfgSections:    make(map[string]ConfigProvider),
		svcRegistry:    make(ServiceRegistry),
		moduleRegistry: make(ModuleRegistry),
		logger:         &logger{t},
	}

	// Register modules with dependencies
	app.RegisterModule(&testModule{name: "database", dependencies: []string{}})
	app.RegisterModule(&testModule{name: "cache", dependencies: []string{}})
	app.RegisterModule(&testModule{name: "api", dependencies: []string{"database", "cache"}})

	// Get all modules for introspection
	modules := app.GetAllModules()

	// Build a map of module info
	moduleInfo := make(map[string][]string)
	for name, mod := range modules {
		// Type assert to DependencyAware to access Dependencies method
		if depAware, ok := mod.(DependencyAware); ok {
			moduleInfo[name] = depAware.Dependencies()
		} else {
			moduleInfo[name] = []string{}
		}
	}

	// Verify we can inspect all modules
	if len(moduleInfo) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(moduleInfo))
	}

	// Verify dependencies
	if len(moduleInfo["database"]) != 0 {
		t.Errorf("database should have no dependencies, got %v", moduleInfo["database"])
	}
	if len(moduleInfo["api"]) != 2 {
		t.Errorf("api should have 2 dependencies, got %v", moduleInfo["api"])
	}
}
