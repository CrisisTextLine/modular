package feeders

import (
	"errors"
	"os"
	"testing"
)

func TestYamlFeeder_Feed_StringConversions(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
intFromString: "42"
floatFromString: "3.14"
boolFromString: "true"
boolFromOne: "1"
boolFromFalse: "false"
boolFromZero: "0"
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	type Config struct {
		IntFromString   int     `yaml:"intFromString"`
		FloatFromString float64 `yaml:"floatFromString"`
		BoolFromString  bool    `yaml:"boolFromString"`
		BoolFromOne     bool    `yaml:"boolFromOne"`
		BoolFromFalse   bool    `yaml:"boolFromFalse"`
		BoolFromZero    bool    `yaml:"boolFromZero"`
	}

	// Test with field tracking enabled to use custom parsing
	tracker := NewDefaultFieldTracker()
	var config Config
	feeder := NewYamlFeeder(tempFile.Name())
	feeder.SetFieldTracker(tracker)
	err = feeder.Feed(&config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.IntFromString != 42 {
		t.Errorf("Expected IntFromString to be 42, got %d", config.IntFromString)
	}
	if config.FloatFromString != 3.14 {
		t.Errorf("Expected FloatFromString to be 3.14, got %f", config.FloatFromString)
	}
	if !config.BoolFromString {
		t.Errorf("Expected BoolFromString to be true, got false")
	}
	if !config.BoolFromOne {
		t.Errorf("Expected BoolFromOne to be true, got false")
	}
	if config.BoolFromFalse {
		t.Errorf("Expected BoolFromFalse to be false, got true")
	}
	if config.BoolFromZero {
		t.Errorf("Expected BoolFromZero to be false, got true")
	}
}

func TestYamlFeeder_Feed_BoolConversionError(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
boolField: "invalid"
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	type Config struct {
		BoolField bool `yaml:"boolField"`
	}

	tracker := NewDefaultFieldTracker()
	var config Config
	feeder := NewYamlFeeder(tempFile.Name())
	feeder.SetFieldTracker(tracker)
	err = feeder.Feed(&config)
	if err == nil {
		t.Error("Expected error for invalid bool conversion")
	}
	if !errors.Is(err, ErrYamlBoolConversion) {
		t.Errorf("Expected ErrYamlBoolConversion, got %v", err)
	}
}

func TestYamlFeeder_Feed_IntConversionError(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	yamlContent := `
intField: "not_a_number"
`
	if _, err := tempFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	type Config struct {
		IntField int `yaml:"intField"`
	}

	tracker := NewDefaultFieldTracker()
	var config Config
	feeder := NewYamlFeeder(tempFile.Name())
	feeder.SetFieldTracker(tracker)
	err = feeder.Feed(&config)
	if err == nil {
		t.Error("Expected error for invalid int conversion")
	}
}
