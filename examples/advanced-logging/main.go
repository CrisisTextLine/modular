package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/CrisisTextLine/modular"
	"github.com/CrisisTextLine/modular/feeders"
	"github.com/CrisisTextLine/modular/modules/chimux"
	"github.com/CrisisTextLine/modular/modules/httpclient"
	"github.com/CrisisTextLine/modular/modules/httpserver"
	"github.com/CrisisTextLine/modular/modules/reverseproxy"
)

type AppConfig struct {
	// Empty config struct for the advanced-logging example
	// Configuration is handled by individual modules
}

func main() {
	// Configure feeders
	modular.ConfigFeeders = []modular.Feeder{
		feeders.NewYamlFeeder("config.yaml"),
		feeders.NewEnvFeeder(),
	}

	// Create a new application
	app := modular.NewStdApplication(
		modular.NewStdConfigProvider(&AppConfig{}),
		slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{},
		)),
	)

	app.Logger().Info("Advanced HTTP Client Logging Demonstration")
	app.Logger().Info("This example demonstrates detailed HTTP request/response logging")
	app.Logger().Info("The server will act as a reverse proxy, making HTTP requests that will be logged")
	app.Logger().Info("Check the ./logs directory for detailed log files")

	// Register the modules in the correct order
	// First the httpclient module with advanced logging
	app.RegisterModule(httpclient.NewHTTPClientModule())

	// Then the modules that depend on it
	app.RegisterModule(chimux.NewChiMuxModule())
	app.RegisterModule(reverseproxy.NewModule())
	app.RegisterModule(httpserver.NewHTTPServerModule())

	// Start the application in background to demonstrate logging
	go func() {
		if err := app.Run(); err != nil {
			app.Logger().Error("Application error", "error", err)
			os.Exit(1)
		}
	}()

	// Give the server time to start
	time.Sleep(3 * time.Second)

	app.Logger().Info("Server started - making test requests to trigger HTTP client logging")
	app.Logger().Info("Access these URLs to see HTTP client logs:")
	app.Logger().Info("  http://localhost:8080/proxy/httpbin/json")
	app.Logger().Info("  http://localhost:8080/proxy/httpbin/user-agent")
	app.Logger().Info("  http://localhost:8080/proxy/httpbin/headers")

	// Make some test requests to demonstrate the logging
	client := &http.Client{Timeout: 10 * time.Second}
	testURLs := []string{
		"http://localhost:8080/proxy/httpbin/json",
		"http://localhost:8080/proxy/httpbin/user-agent",
		"http://localhost:8080/proxy/httpbin/headers",
	}

	for _, url := range testURLs {
		app.Logger().Info("Making test request", "url", url)
		resp, err := client.Get(url)
		if err != nil {
			app.Logger().Error("Request failed", "url", url, "error", err)
			continue
		}
		resp.Body.Close()
		app.Logger().Info("Request completed", "url", url, "status", resp.Status)
		time.Sleep(2 * time.Second)
	}

	app.Logger().Info("Advanced logging demonstration complete")
	app.Logger().Info("Check the ./logs directory for detailed HTTP client logs")
	app.Logger().Info("The logs contain request headers, response headers, and body content")

	// Keep running for a bit longer to allow manual testing
	// In CI environments, run for a shorter time
	duration := 30 * time.Second
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		duration = 4 * time.Second
	}
	app.Logger().Info("Server will continue running for manual testing...", "duration", duration)
	time.Sleep(duration)
}
