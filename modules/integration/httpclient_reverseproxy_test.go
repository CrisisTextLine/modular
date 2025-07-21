package integration

import (
	"net/http"
	"testing"

	"github.com/CrisisTextLine/modular"
	"github.com/CrisisTextLine/modular/modules/httpclient"
	"github.com/CrisisTextLine/modular/modules/reverseproxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPClientReverseProxyIntegration tests that reverseproxy module properly
// uses the httpclient module when both are present in the same application
func TestHTTPClientReverseProxyIntegration(t *testing.T) {
	// Create application
	app := modular.NewStdApplication(modular.NewStdConfigProvider(nil), &testLogger{t: t})

	// Register httpclient module
	httpClientModule := httpclient.NewHTTPClientModule()
	app.RegisterModule(httpClientModule)

	// Create a mock router that satisfies the routerService interface
	mockRouter := &testRouter{
		routes: make(map[string]http.HandlerFunc),
	}
	
	// Register router service manually for testing
	err := app.RegisterService("router", mockRouter)
	require.NoError(t, err)

	// Register reverseproxy module
	reverseProxyModule := reverseproxy.NewModule()
	app.RegisterModule(reverseProxyModule)

	// Initialize application
	err = app.Init()
	require.NoError(t, err)

	// Verify that the reverseproxy module is using an HTTP client
	// (we can't directly compare instances since they go through the constructor injection,
	// but we can verify that it's not nil and that the httpclient module provided services)
	// Note: We can't access the httpClient directly since it's a private field,
	// but we can verify the service resolution worked correctly.
	
	// Verify that httpclient module provides the expected services
	httpClientServiceAware, ok := httpClientModule.(modular.ServiceAware)
	require.True(t, ok, "httpclient should be ServiceAware")

	providedServices := httpClientServiceAware.ProvidesServices()
	require.Len(t, providedServices, 3, "httpclient should provide 3 services")

	// Verify service names
	serviceNames := make(map[string]bool)
	for _, svc := range providedServices {
		serviceNames[svc.Name] = true
	}
	assert.True(t, serviceNames["httpclient"], "should provide 'httpclient' service")
	assert.True(t, serviceNames["httpclient-service"], "should provide 'httpclient-service' service")
	assert.True(t, serviceNames["http-doer"], "should provide 'http-doer' service")

	// Verify that reverseproxy module declares the correct dependencies
	var reverseProxyServiceAware modular.ServiceAware = reverseProxyModule
	require.NotNil(t, reverseProxyServiceAware, "reverseproxy should be ServiceAware")

	requiredServices := reverseProxyServiceAware.RequiresServices()
	require.Len(t, requiredServices, 3, "reverseproxy should require 3 services")

	// Map dependencies by name
	depMap := make(map[string]modular.ServiceDependency)
	for _, dep := range requiredServices {
		depMap[dep.Name] = dep
	}

	// Verify the dependencies are declared correctly
	assert.Contains(t, depMap, "router", "should declare router dependency")
	assert.Contains(t, depMap, "httpclient", "should declare httpclient dependency (name-based)")
	assert.Contains(t, depMap, "http-doer", "should declare http-doer dependency (interface-based)")
}

// testRouter implements the routerService interface for testing
type testRouter struct {
	routes map[string]http.HandlerFunc
}

func (tr *testRouter) Handle(pattern string, handler http.Handler) {
	tr.routes[pattern] = handler.ServeHTTP
}

func (tr *testRouter) HandleFunc(pattern string, handler http.HandlerFunc) {
	tr.routes[pattern] = handler
}

func (tr *testRouter) Mount(pattern string, h http.Handler) {
	tr.routes[pattern] = h.ServeHTTP
}

func (tr *testRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	// No-op for test router
}

func (tr *testRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := tr.routes[r.URL.Path]; ok {
		handler(w, r)
		return
	}
	if handler, ok := tr.routes["/*"]; ok {
		handler(w, r)
		return
	}
	http.NotFound(w, r)
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