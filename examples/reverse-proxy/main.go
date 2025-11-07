package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/CrisisTextLine/modular"
	"github.com/CrisisTextLine/modular/feeders"
	"github.com/CrisisTextLine/modular/modules/chimux"
	"github.com/CrisisTextLine/modular/modules/httpserver"
	"github.com/CrisisTextLine/modular/modules/reverseproxy"
)

type AppConfig struct {
	// Empty config struct for the reverse proxy example
	// Configuration is handled by individual modules
}

func main() {
	// Start mock backend servers
	startMockBackends()

	// Create a new application and set feeders per instance (no global mutation)
	app := modular.NewStdApplication(
		modular.NewStdConfigProvider(&AppConfig{}),
		slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug},
		)),
	)
	if stdApp, ok := app.(*modular.StdApplication); ok {
		stdApp.SetConfigFeeders([]modular.Feeder{
			feeders.NewYamlFeeder("config.yaml"),
			feeders.NewEnvFeeder(),
		})
	}

	// Create tenant service
	tenantService := modular.NewStandardTenantService(app.Logger())
	if err := app.RegisterService("tenantService", tenantService); err != nil {
		app.Logger().Error("Failed to register tenant service", "error", err)
		os.Exit(1)
	}

	// Register tenants with their configurations
	err := tenantService.RegisterTenant("tenant1", map[string]modular.ConfigProvider{
		"reverseproxy": modular.NewStdConfigProvider(&reverseproxy.ReverseProxyConfig{
			DefaultBackend: "tenant1-backend",
			BackendServices: map[string]string{
				"tenant1-backend": "http://localhost:9002",
			},
		}),
	})
	if err != nil {
		app.Logger().Error("Failed to register tenant1", "error", err)
		os.Exit(1)
	}

	err = tenantService.RegisterTenant("tenant2", map[string]modular.ConfigProvider{
		"reverseproxy": modular.NewStdConfigProvider(&reverseproxy.ReverseProxyConfig{
			DefaultBackend: "tenant2-backend",
			BackendServices: map[string]string{
				"tenant2-backend": "http://localhost:9003",
			},
		}),
	})
	if err != nil {
		app.Logger().Error("Failed to register tenant2", "error", err)
		os.Exit(1)
	}

	// Register the modules in dependency order
	app.RegisterModule(chimux.NewChiMuxModule())
	
	// Create reverse proxy module and configure dynamic response header modification
	proxyModule := reverseproxy.NewModule()
	
	// Set a custom response header modifier to demonstrate dynamic CORS header consolidation
	proxyModule.SetResponseHeaderModifier(func(resp *http.Response, backendID string, tenantID modular.TenantID) error {
		// Add custom headers based on backend and tenant
		resp.Header.Set("X-Backend-Served-By", backendID)
		if tenantID != "" {
			resp.Header.Set("X-Tenant-Served", string(tenantID))
		}
		
		// Example: Dynamically set Cache-Control based on status code
		if resp.StatusCode == http.StatusOK {
			resp.Header.Set("Cache-Control", "public, max-age=300")
		} else {
			resp.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		
		return nil
	})
	
	app.RegisterModule(proxyModule)
	app.RegisterModule(httpserver.NewHTTPServerModule())

	// Run application with lifecycle management
	if err := app.Run(); err != nil {
		app.Logger().Error("Application error", "error", err)
		os.Exit(1)
	}
}

// startMockBackends starts mock backend servers on different ports
func startMockBackends() {
	// Global default backend (port 9001)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"backend":"global-default","path":"%s","method":"%s"}`, r.URL.Path, r.Method)
		})
		fmt.Println("Starting global-default backend on :9001")
		if err := http.ListenAndServe(":9001", mux); err != nil { //nolint:gosec
			fmt.Printf("Backend server error on :9001: %v\n", err)
		}
	}()

	// Tenant1 backend (port 9002)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"backend":"tenant1-backend","path":"%s","method":"%s"}`, r.URL.Path, r.Method)
		})
		fmt.Println("Starting tenant1-backend on :9002")
		if err := http.ListenAndServe(":9002", mux); err != nil { //nolint:gosec
			fmt.Printf("Backend server error on :9002: %v\n", err)
		}
	}()

	// Tenant2 backend (port 9003)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"backend":"tenant2-backend","path":"%s","method":"%s"}`, r.URL.Path, r.Method)
		})
		fmt.Println("Starting tenant2-backend on :9003")
		if err := http.ListenAndServe(":9003", mux); err != nil { //nolint:gosec
			fmt.Printf("Backend server error on :9003: %v\n", err)
		}
	}()

	// Specific API backend (port 9004) - simulates a backend with CORS headers
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Backend sets its own CORS headers (which will be overridden by proxy)
			w.Header().Set("Access-Control-Allow-Origin", "http://old-domain.com")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("X-Internal-Header", "internal-value")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"backend":"specific-api","path":"%s","method":"%s","note":"backend CORS headers will be overridden"}`, r.URL.Path, r.Method)
		})
		fmt.Println("Starting specific-api backend on :9004")
		if err := http.ListenAndServe(":9004", mux); err != nil { //nolint:gosec
			fmt.Printf("Backend server error on :9004: %v\n", err)
		}
	}()
}
