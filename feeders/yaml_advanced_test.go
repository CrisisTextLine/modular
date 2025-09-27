package feeders

import (
	"os"
	"strings"
	"testing"
)

func TestYamlFeeder_Feed_MapFields(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
connections:
  primary:
    host: "localhost"
    port: 5432
    database: "mydb"
  secondary:
    host: "backup.host"
    port: 5433
    database: "backupdb"
stringMap:
  key1: "value1"
  key2: "value2"
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	type DBConnection struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Database string `yaml:"database"`
	}

	type Config struct {
		Connections map[string]DBConnection `yaml:"connections"`
		StringMap   map[string]string       `yaml:"stringMap"`
	}

	var config Config
	feeder := NewYamlFeeder(tempFile.Name())
	err = feeder.Feed(&config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(config.Connections))
	}
	if config.Connections["primary"].Host != "localhost" {
		t.Errorf("Expected primary host to be 'localhost', got '%s'", config.Connections["primary"].Host)
	}
	if config.Connections["primary"].Port != 5432 {
		t.Errorf("Expected primary port to be 5432, got %d", config.Connections["primary"].Port)
	}
	if config.Connections["secondary"].Database != "backupdb" {
		t.Errorf("Expected secondary database to be 'backupdb', got '%s'", config.Connections["secondary"].Database)
	}

	if len(config.StringMap) != 2 {
		t.Errorf("Expected 2 string map entries, got %d", len(config.StringMap))
	}
	if config.StringMap["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got '%s'", config.StringMap["key1"])
	}
}

func TestYamlFeeder_Feed_FieldTracking(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
app:
  name: TestApp
  version: "1.0"
database:
  host: localhost
  port: 5432
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	type Config struct {
		App struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		} `yaml:"app"`
		Database struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
		} `yaml:"database"`
		NotFound string `yaml:"notfound"`
	}

	tracker := NewDefaultFieldTracker()
	var config Config
	feeder := NewYamlFeeder(tempFile.Name())
	feeder.SetFieldTracker(tracker)
	err = feeder.Feed(&config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	populations := tracker.GetFieldPopulations()
	if len(populations) == 0 {
		t.Error("Expected field populations to be recorded")
	}

	// Check that we have records for found fields
	foundFields := make(map[string]bool)
	for _, pop := range populations {
		if pop.FoundKey != "" {
			foundFields[pop.FieldPath] = true
		}
	}

	expectedFields := []string{"App.Name", "App.Version", "Database.Host", "Database.Port"}
	for _, field := range expectedFields {
		if !foundFields[field] {
			t.Errorf("Expected field %s to be found and recorded", field)
		}
	}
}

func TestYamlFeeder_Feed_VerboseDebug(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
app:
  name: TestApp
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	type Config struct {
		App struct {
			Name string `yaml:"name"`
		} `yaml:"app"`
	}

	logger := &mockLogger{}
	var config Config
	feeder := NewYamlFeeder(tempFile.Name())
	feeder.SetVerboseDebug(true, logger)
	err = feeder.Feed(&config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	messages := logger.getMessages()
	if len(messages) == 0 {
		t.Error("Expected debug messages to be logged")
	}

	// Check for specific debug messages
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "Starting feed process") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'Starting feed process' in debug messages")
	}
}

func TestYamlFeeder_Feed_NoFieldTracker(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
app:
  name: TestApp
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	type Config struct {
		App struct {
			Name string `yaml:"name"`
		} `yaml:"app"`
	}

	var config Config
	feeder := NewYamlFeeder(tempFile.Name())
	// Don't set field tracker - should use original behavior
	err = feeder.Feed(&config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.App.Name != "TestApp" {
		t.Errorf("Expected Name to be 'TestApp', got '%s'", config.App.Name)
	}
}

func TestYamlFeeder_Feed_NonStructPointer(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
- item1
- item2
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	tracker := NewDefaultFieldTracker()
	var config []string
	feeder := NewYamlFeeder(tempFile.Name())
	feeder.SetFieldTracker(tracker)
	err = feeder.Feed(&config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config) != 2 {
		t.Errorf("Expected 2 items, got %d", len(config))
	}
}
