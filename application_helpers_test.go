package modular

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// testCfg for basic configuration testing
type testCfg struct {
	Str string `yaml:"str"`
	Num int    `yaml:"num"`
}

// logger for testing with caller information
type logger struct {
	t *testing.T
}

func (l *logger) getCallerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	relPath, err := filepath.Rel(wd, file)
	if err != nil {
		relPath = file
	}
	return fmt.Sprintf("%s:%d", relPath, line)
}

func (l *logger) Info(msg string, args ...any) {
	dir := l.getCallerInfo()
	l.t.Log(fmt.Sprintf("[%s] %s", dir, msg), args)
}

func (l *logger) Error(msg string, args ...any) {
	dir := l.getCallerInfo()
	l.t.Error(fmt.Sprintf("[%s] %s", dir, msg), args)
}

func (l *logger) Warn(msg string, args ...any) {
	dir := l.getCallerInfo()
	l.t.Log(fmt.Sprintf("[%s] %s", dir, msg), args)
}

func (l *logger) Debug(msg string, args ...any) {
	dir := l.getCallerInfo()
	l.t.Log(fmt.Sprintf("[%s] %s", dir, msg), args)
}

// initTestLogger for debug module tests
type initTestLogger struct {
	t *testing.T
}

func (l *initTestLogger) Info(msg string, args ...any) {
	if l.t != nil {
		l.t.Logf("[INFO] %s", msg)
	}
}

func (l *initTestLogger) Error(msg string, args ...any) {
	if l.t != nil {
		l.t.Logf("[ERROR] %s", msg)
	}
}

func (l *initTestLogger) Warn(msg string, args ...any) {
	if l.t != nil {
		l.t.Logf("[WARN] %s", msg)
	}
}

func (l *initTestLogger) Debug(msg string, args ...any) {
	if l.t != nil {
		l.t.Logf("[DEBUG] %s", msg)
	}
}

// Helper function for testing AppConfigLoader
func testAppConfigLoader(app *StdApplication) error {
	// Return error if config provider is nil
	if app.cfgProvider == nil {
		return ErrConfigProviderNil
	}

	// Return error if there's an "error-trigger" section
	if _, exists := app.cfgSections["error-trigger"]; exists {
		return ErrConfigSectionError
	}

	return nil
}

// Define test service interfaces and implementations
type StorageService interface {
	Get(key string) string
}

type MockStorage struct {
	data map[string]string
}

func (m *MockStorage) Get(key string) string {
	return m.data[key]
}

// Create mock module implementation for testing
type testModule struct {
	name         string
	dependencies []string
}

// Implement Module interface for our test module
func (m testModule) Name() string                          { return m.name }
func (m testModule) Dependencies() []string                { return m.dependencies }
func (m testModule) Init(Application) error                { return nil }
func (m testModule) Start(context.Context) error           { return nil }
func (m testModule) Stop(context.Context) error            { return nil }
func (m testModule) RegisterConfig(Application) error      { return nil }
func (m testModule) ProvidesServices() []ServiceProvider   { return nil }
func (m testModule) RequiresServices() []ServiceDependency { return nil }

// Mock module for testing configuration registration
type configRegisteringModule struct {
	testModule
	configRegistered bool
	initCalled       bool
	initError        error
}

func (m *configRegisteringModule) RegisterConfig(app Application) error {
	app.RegisterConfigSection(m.name+"-config", NewStdConfigProvider(m.name+"-config-value"))
	m.configRegistered = true
	return nil
}

func (m *configRegisteringModule) Init(Application) error {
	m.initCalled = true
	return m.initError
}

// Mock module that provides services
type serviceProvidingModule struct {
	testModule
	services []ServiceProvider
}

func (m *serviceProvidingModule) ProvidesServices() []ServiceProvider {
	return m.services
}

// Mock module that tracks lifecycle methods
type lifecycleTestModule struct {
	testModule
	initCalled  bool
	startCalled bool
	stopCalled  bool
	startError  error
	stopError   error
}

func (m *lifecycleTestModule) Init(Application) error {
	m.initCalled = true
	return nil
}

func (m *lifecycleTestModule) Start(context.Context) error {
	m.startCalled = true
	return m.startError
}

func (m *lifecycleTestModule) Stop(context.Context) error {
	m.stopCalled = true
	return m.stopError
}

// Helper for error checking
func IsServiceAlreadyRegisteredError(err error) bool {
	return err != nil && ErrorIs(err, ErrServiceAlreadyRegistered)
}

func IsServiceNotFoundError(err error) bool {
	return err != nil && ErrorIs(err, ErrServiceNotFound)
}

func IsServiceIncompatibleError(err error) bool {
	return err != nil && ErrorIs(err, ErrServiceIncompatible)
}

func IsCircularDependencyError(err error) bool {
	return err != nil && ErrorIs(err, ErrCircularDependency)
}

func IsModuleDependencyMissingError(err error) bool {
	return err != nil && ErrorIs(err, ErrModuleDependencyMissing)
}

// ErrorIs is a helper function that checks if err contains target error
func ErrorIs(err, target error) bool {
	// Simple implementation that checks if target is in err's chain
	for {
		if errors.Is(err, target) {
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
			if err == nil {
				return false
			}
		} else {
			return false
		}
	}
}

// Placeholder errors for tests
var (
	ErrModuleStartFailed = fmt.Errorf("module start failed")
	ErrModuleStopFailed  = fmt.Errorf("module stop failed")
)
