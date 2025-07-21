package reverseproxy

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/CrisisTextLine/modular"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReverseProxyServiceDependencyResolution tests that the reverseproxy module
// can receive HTTP client services both by name and by interface
func TestReverseProxyServiceDependencyResolution(t *testing.T) {
	// Test 1: Name-based service resolution
	t.Run("NameBasedServiceResolution", func(t *testing.T) {
		app := modular.NewStdApplication(modular.NewStdConfigProvider(nil), &testLogger{t: t})

		// Create mock HTTP client
		mockClient := &http.Client{}

		// Create a mock router service that satisfies the routerService interface
		mockRouter := &testRouter{
			routes: make(map[string]http.HandlerFunc),
		}

		// Register services manually for testing
		err := app.RegisterService("router", mockRouter)
		require.NoError(t, err)

		err = app.RegisterService("httpclient", mockClient)
		require.NoError(t, err)

		// Create reverseproxy module
		reverseProxyModule := NewModule()
		app.RegisterModule(reverseProxyModule)

		// Initialize application
		err = app.Init()
		require.NoError(t, err)

		// Verify the module received the httpclient service
		assert.NotNil(t, reverseProxyModule.httpClient, "HTTP client should be set")
		assert.Same(t, mockClient, reverseProxyModule.httpClient, "Should use the provided HTTP client")
	})

	// Test 2: Interface-based service resolution
	t.Run("InterfaceBasedServiceResolution", func(t *testing.T) {
		app := modular.NewStdApplication(modular.NewStdConfigProvider(nil), &testLogger{t: t})

		// Create mock HTTP client
		mockClient := &http.Client{}

		// Create a mock router service that satisfies the routerService interface
		mockRouter := &testRouter{
			routes: make(map[string]http.HandlerFunc),
		}

		// Register services manually for testing
		err := app.RegisterService("router", mockRouter)
		require.NoError(t, err)

		// Register the HTTP client as an httpDoer interface (not by name "httpclient")
		err = app.RegisterService("http-doer", mockClient)
		require.NoError(t, err)

		// Create reverseproxy module
		reverseProxyModule := NewModule()
		app.RegisterModule(reverseProxyModule)

		// Initialize application
		err = app.Init()
		require.NoError(t, err)

		// Verify the module received the http-doer service
		assert.NotNil(t, reverseProxyModule.httpClient, "HTTP client should be set")
		assert.Same(t, mockClient, reverseProxyModule.httpClient, "Should use the provided HTTP client via interface")
	})

	// Test 3: No HTTP client service (default client creation)
	t.Run("DefaultClientCreation", func(t *testing.T) {
		app := modular.NewStdApplication(modular.NewStdConfigProvider(nil), &testLogger{t: t})

		// Create a mock router service that satisfies the routerService interface
		mockRouter := &testRouter{
			routes: make(map[string]http.HandlerFunc),
		}

		// Register only router service, no HTTP client services
		err := app.RegisterService("router", mockRouter)
		require.NoError(t, err)

		// Create reverseproxy module
		reverseProxyModule := NewModule()
		app.RegisterModule(reverseProxyModule)

		// Initialize application
		err = app.Init()
		require.NoError(t, err)

		// Verify the module created a default HTTP client
		assert.NotNil(t, reverseProxyModule.httpClient, "HTTP client should be created as default")
	})
}

// TestHTTPDoerInterfaceImplementation tests that http.Client implements the httpDoer interface
func TestHTTPDoerInterfaceImplementation(t *testing.T) {
	client := &http.Client{}

	// Test that http.Client implements httpDoer interface
	var doer httpDoer = client
	assert.NotNil(t, doer, "http.Client should implement httpDoer interface")

	// Test reflection-based interface checking (this is what the framework uses)
	clientType := reflect.TypeOf(client)
	doerInterface := reflect.TypeOf((*httpDoer)(nil)).Elem()

	assert.True(t, clientType.Implements(doerInterface), 
		"http.Client should implement httpDoer interface via reflection")
}

// TestServiceDependencyConfiguration tests that the reverseproxy module declares the correct dependencies
func TestServiceDependencyConfiguration(t *testing.T) {
	module := NewModule()

	// Check that module implements ServiceAware
	var serviceAware modular.ServiceAware = module
	require.NotNil(t, serviceAware, "reverseproxy module should implement ServiceAware")

	// Get service dependencies
	dependencies := serviceAware.RequiresServices()
	require.Len(t, dependencies, 3, "reverseproxy should declare 3 service dependencies")

	// Map dependencies by name for easy checking
	depMap := make(map[string]modular.ServiceDependency)
	for _, dep := range dependencies {
		depMap[dep.Name] = dep
	}

	// Check router dependency (required, interface-based)
	routerDep, exists := depMap["router"]
	assert.True(t, exists, "router dependency should exist")
	assert.True(t, routerDep.Required, "router dependency should be required")
	assert.True(t, routerDep.MatchByInterface, "router dependency should use interface matching")

	// Check httpclient dependency (optional, name-based)
	httpclientDep, exists := depMap["httpclient"]
	assert.True(t, exists, "httpclient dependency should exist")
	assert.False(t, httpclientDep.Required, "httpclient dependency should be optional")
	assert.False(t, httpclientDep.MatchByInterface, "httpclient dependency should use name matching")

	// Check http-doer dependency (optional, interface-based)
	doerDep, exists := depMap["http-doer"]
	assert.True(t, exists, "http-doer dependency should exist")
	assert.False(t, doerDep.Required, "http-doer dependency should be optional")
	assert.True(t, doerDep.MatchByInterface, "http-doer dependency should use interface matching")
	assert.NotNil(t, doerDep.SatisfiesInterface, "http-doer dependency should specify interface")
}

// testLogger is a simple test logger implementation
type testLogger struct {
	t *testing.T
}

func (l *testLogger) Debug(msg string, keyvals ...interface{}) {
	l.t.Logf("DEBUG: %s %v", msg, keyvals)
}

func (l *testLogger) Info(msg string, keyvals ...interface{}) {
	l.t.Logf("INFO: %s %v", msg, keyvals)
}

func (l *testLogger) Warn(msg string, keyvals ...interface{}) {
	l.t.Logf("WARN: %s %v", msg, keyvals)
}

func (l *testLogger) Error(msg string, keyvals ...interface{}) {
	l.t.Logf("ERROR: %s %v", msg, keyvals)
}