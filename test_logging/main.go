package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/CrisisTextLine/modular"
	"github.com/CrisisTextLine/modular/modules/httpclient"
)

// Simple logger that captures output
type TestLogger struct {
	logs []string
}

func (l *TestLogger) Debug(msg string, keyvals ...interface{}) {
	logEntry := fmt.Sprintf("DEBUG: %s", msg)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			logEntry += fmt.Sprintf(" [%v=%v]", keyvals[i], keyvals[i+1])
		}
	}
	l.logs = append(l.logs, logEntry)
	fmt.Println(logEntry)
}

func (l *TestLogger) Info(msg string, keyvals ...interface{}) {
	logEntry := fmt.Sprintf("INFO: %s", msg)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			logEntry += fmt.Sprintf(" [%v=%v]", keyvals[i], keyvals[i+1])
		}
	}
	l.logs = append(l.logs, logEntry)
	fmt.Println(logEntry)
}

func (l *TestLogger) Warn(msg string, keyvals ...interface{}) {
	logEntry := fmt.Sprintf("WARN: %s", msg)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			logEntry += fmt.Sprintf(" [%v=%v]", keyvals[i], keyvals[i+1])
		}
	}
	l.logs = append(l.logs, logEntry)
	fmt.Println(logEntry)
}

func (l *TestLogger) Error(msg string, keyvals ...interface{}) {
	logEntry := fmt.Sprintf("ERROR: %s", msg)
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			logEntry += fmt.Sprintf(" [%v=%v]", keyvals[i], keyvals[i+1])
		}
	}
	l.logs = append(l.logs, logEntry)
	fmt.Println(logEntry)
}

// Simple application mock
type TestApp struct {
	logger        modular.Logger
	configSection *httpclient.Config
}

func (a *TestApp) Logger() modular.Logger {
	return a.logger
}

func (a *TestApp) GetConfigSection(name string) (modular.ConfigProvider, error) {
	return &TestConfigProvider{config: a.configSection}, nil
}

func (a *TestApp) RegisterConfigSection(name string, provider modular.ConfigProvider) {
	// No-op
}

// Satisfy other required methods with no-ops
func (a *TestApp) Name() string                                        { return "test-app" }
func (a *TestApp) IsInitializing() bool                                { return false }
func (a *TestApp) IsStarting() bool                                    { return false }
func (a *TestApp) IsStopping() bool                                    { return false }
func (a *TestApp) RegisterModule(module modular.Module)                {}
func (a *TestApp) GetModuleByName(name string) (modular.Module, error) { return nil, nil }
func (a *TestApp) GetAllModules() []modular.Module                     { return nil }
func (a *TestApp) Run() error                                          { return nil }
func (a *TestApp) Shutdown(ctx context.Context) error                  { return nil }
func (a *TestApp) Init() error                                         { return nil }
func (a *TestApp) Start() error                                        { return nil }
func (a *TestApp) Stop() error                                         { return nil }
func (a *TestApp) SetLogger(logger modular.Logger)                     {}
func (a *TestApp) ConfigProvider() modular.ConfigProvider              { return nil }
func (a *TestApp) SvcRegistry() modular.ServiceRegistry                { return nil }
func (a *TestApp) ConfigSections() map[string]modular.ConfigProvider   { return nil }
func (a *TestApp) RegisterService(name string, service any) error      { return nil }
func (a *TestApp) GetService(name string, target any) error            { return nil }
func (a *TestApp) IsVerboseConfig() bool                               { return false }
func (a *TestApp) SetVerboseConfig(verbose bool)                       {}

type TestConfigProvider struct {
	config *httpclient.Config
}

func (c *TestConfigProvider) GetConfig() interface{} {
	return c.config
}

func main() {
	// Create test logger
	logger := &TestLogger{}

	// Create httpclient config with verbose logging enabled but very small body log size
	config := &httpclient.Config{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     30,
		RequestTimeout:      10,
		TLSTimeout:          5,
		DisableCompression:  false,
		DisableKeepAlives:   false,
		Verbose:             true,
		VerboseOptions: &httpclient.VerboseOptions{
			LogHeaders:     true,
			LogBody:        true,
			MaxBodyLogSize: 0,     // Very small size to trigger truncation
			LogToFile:      false, // Log to application logger instead of files
		},
	}

	// Create app with config
	app := &TestApp{
		logger:        logger,
		configSection: config,
	}

	// Create and initialize the module
	module := httpclient.NewHTTPClientModule()
	err := module.Init(app)
	if err != nil {
		fmt.Printf("Failed to initialize module: %v\n", err)
		os.Exit(1)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Hello, World!", "status": "success"}`))
	}))
	defer server.Close()

	// Get the client
	clientService := module.(httpclient.ClientService)
	client := clientService.Client()

	// Make a request with some body
	jsonBody := bytes.NewBufferString(`{"request": "test", "data": "sample"}`)
	req, err := http.NewRequest("POST", server.URL+"/api/test", jsonBody)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	fmt.Println("\n=== Making HTTP Request ===")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("\n=== Response Status: %s ===\n", resp.Status)

	// Check if we got useful logs
	fmt.Println("\n=== Analyzing Logs ===")
	foundRequestDump := false
	foundResponseDump := false
	
	for _, log := range logger.logs {
		if strings.Contains(log, "Request dump") {
			foundRequestDump = true
			fmt.Printf("REQUEST DUMP LOG: %s\n", log)
		}
		if strings.Contains(log, "Response dump") {
			foundResponseDump = true
			fmt.Printf("RESPONSE DUMP LOG: %s\n", log)
		}
	}
	
	if !foundRequestDump {
		fmt.Println("❌ No request dump found in logs!")
	}
	if !foundResponseDump {
		fmt.Println("❌ No response dump found in logs!")
	}
	
	if foundRequestDump && foundResponseDump {
		fmt.Println("✅ Found both request and response dumps")
	}
}