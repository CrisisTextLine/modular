package modular

import (
	"context"
	"fmt"
	"testing"
)

// ExampleApplication_GetModule demonstrates basic usage of GetModule
func ExampleApplication_GetModule() {
	// Create application
	app := NewStdApplication(
		NewStdConfigProvider(struct{}{}),
		&exampleLogger{},
	)

	// Register some modules
	app.RegisterModule(&exampleDatabaseModule{})
	app.RegisterModule(&exampleAPIModule{})

	// Get a specific module by name
	dbModule := app.GetModule("database")
	if dbModule != nil {
		fmt.Println("Found database module:", dbModule.Name())
	}

	// Type assert to access module-specific methods
	if db, ok := dbModule.(*exampleDatabaseModule); ok {
		fmt.Println("Database module is ready:", db.IsReady())
	}

	// Check for optional module
	cacheModule := app.GetModule("cache")
	if cacheModule == nil {
		fmt.Println("Cache module not loaded (optional)")
	}

	// Output:
	// Found database module: database
	// Database module is ready: true
	// Cache module not loaded (optional)
}

// ExampleApplication_GetAllModules demonstrates introspection of all modules
func ExampleApplication_GetAllModules() {
	// Create application
	app := NewStdApplication(
		NewStdConfigProvider(struct{}{}),
		&exampleLogger{},
	)

	// Register modules
	app.RegisterModule(&exampleDatabaseModule{})
	app.RegisterModule(&exampleAPIModule{})
	app.RegisterModule(&exampleWorkerModule{})

	// Get all modules for introspection
	modules := app.GetAllModules()

	fmt.Println("Registered modules:")
	for name := range modules {
		fmt.Println("-", name)
	}

	// Count modules
	fmt.Printf("Total: %d modules\n", len(modules))

	// Output:
	// Registered modules:
	// - database
	// - api
	// - worker
	// Total: 3 modules
}

// ExampleApplication_GetModule_optionalDependency shows checking for optional module dependencies
func ExampleApplication_GetModule_optionalDependency() {
	// Create application
	app := NewStdApplication(
		NewStdConfigProvider(struct{}{}),
		&exampleLogger{},
	)

	// Create a module that has optional dependencies
	apiModule := &exampleAPIModule{}
	app.RegisterModule(apiModule)

	// Register database module (required)
	app.RegisterModule(&exampleDatabaseModule{})

	// Don't register cache module (optional)

	// In the module's Init, check for optional dependencies
	if app.GetModule("cache") != nil {
		fmt.Println("Cache module available - enabling advanced features")
	} else {
		fmt.Println("Cache module not available - using basic features")
	}

	// Output:
	// Cache module not available - using basic features
}

// Test helper modules for examples

type exampleDatabaseModule struct{}

func (m *exampleDatabaseModule) Name() string { return "database" }
func (m *exampleDatabaseModule) Init(Application) error {
	return nil
}
func (m *exampleDatabaseModule) Dependencies() []string { return nil }
func (m *exampleDatabaseModule) IsReady() bool          { return true }

type exampleAPIModule struct{}

func (m *exampleAPIModule) Name() string                { return "api" }
func (m *exampleAPIModule) Init(Application) error      { return nil }
func (m *exampleAPIModule) Dependencies() []string      { return []string{"database"} }
func (m *exampleAPIModule) Start(context.Context) error { return nil }

type exampleWorkerModule struct{}

func (m *exampleWorkerModule) Name() string           { return "worker" }
func (m *exampleWorkerModule) Init(Application) error { return nil }
func (m *exampleWorkerModule) Dependencies() []string { return nil }

type exampleLogger struct{}

func (l *exampleLogger) Info(msg string, args ...any)  {}
func (l *exampleLogger) Error(msg string, args ...any) {}
func (l *exampleLogger) Warn(msg string, args ...any)  {}
func (l *exampleLogger) Debug(msg string, args ...any) {}

// Test that examples work correctly
func TestExamples(t *testing.T) {
	// Just verify examples compile and run without panicking
	t.Run("GetModule", func(t *testing.T) {
		ExampleApplication_GetModule()
	})

	t.Run("GetAllModules", func(t *testing.T) {
		ExampleApplication_GetAllModules()
	})

	t.Run("OptionalDependency", func(t *testing.T) {
		ExampleApplication_GetModule_optionalDependency()
	})
}
