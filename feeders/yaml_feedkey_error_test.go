package feeders

import (
	"os"
	"testing"
)

func TestYamlFeeder_FeedKey(t *testing.T) {
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

	type AppConfig struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	var appConfig AppConfig
	feeder := NewYamlFeeder(tempFile.Name())
	err = feeder.FeedKey("app", &appConfig)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if appConfig.Name != "TestApp" {
		t.Errorf("Expected Name to be 'TestApp', got '%s'", appConfig.Name)
	}
	if appConfig.Version != "1.0" {
		t.Errorf("Expected Version to be '1.0', got '%s'", appConfig.Version)
	}
}

func TestYamlFeeder_FeedKey_NotFound(t *testing.T) {
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

	type NotFoundConfig struct {
		Value string `yaml:"value"`
	}

	var config NotFoundConfig
	feeder := NewYamlFeeder(tempFile.Name())
	err = feeder.FeedKey("notfound", &config)
	if err != nil {
		t.Fatalf("Expected no error for missing key, got %v", err)
	}

	if config.Value != "" {
		t.Errorf("Expected empty value for missing key, got '%s'", config.Value)
	}
}

func TestYamlFeeder_Feed_FileNotFound(t *testing.T) {
	feeder := NewYamlFeeder("/nonexistent/file.yaml")
	var config struct{}
	err := feeder.Feed(&config)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestYamlFeeder_Feed_InvalidYaml(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	invalidYaml := `
app:
  name: TestApp
  version: "1.0"
  invalid: [unclosed array
`
	if _, err := tempFile.Write([]byte(invalidYaml)); err != nil {
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
	err = feeder.Feed(&config)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
