package feeders

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// verifyOmitemptyTestConfig is a helper function to validate the populated config against expectations
func verifyOmitemptyTestConfig(t *testing.T, feederType string, config *OmitemptyTestConfig, expected map[string]interface{}) {
	t.Helper()

	// Check required fields
	if val, exists := expected["RequiredString"]; exists {
		if config.RequiredString != val.(string) {
			t.Errorf("[%s] RequiredString: expected %q, got %q", feederType, val.(string), config.RequiredString)
		}
	}

	if val, exists := expected["RequiredInt"]; exists {
		if config.RequiredInt != val.(int) {
			t.Errorf("[%s] RequiredInt: expected %d, got %d", feederType, val.(int), config.RequiredInt)
		}
	}

	// Check optional fields with omitempty
	if val, exists := expected["OptionalString"]; exists {
		if config.OptionalString != val.(string) {
			t.Errorf("[%s] OptionalString: expected %q, got %q", feederType, val.(string), config.OptionalString)
		}
	}

	if val, exists := expected["OptionalInt"]; exists {
		if config.OptionalInt != val.(int) {
			t.Errorf("[%s] OptionalInt: expected %d, got %d", feederType, val.(int), config.OptionalInt)
		}
	}

	if val, exists := expected["OptionalBool"]; exists {
		if config.OptionalBool != val.(bool) {
			t.Errorf("[%s] OptionalBool: expected %v, got %v", feederType, val.(bool), config.OptionalBool)
		}
	}

	if val, exists := expected["OptionalFloat64"]; exists {
		if config.OptionalFloat64 != val.(float64) {
			t.Errorf("[%s] OptionalFloat64: expected %f, got %f", feederType, val.(float64), config.OptionalFloat64)
		}
	}

	// Check pointer fields
	if val, exists := expected["OptionalStringPtr"]; exists {
		if val == nil {
			if config.OptionalStringPtr != nil {
				t.Errorf("[%s] OptionalStringPtr: expected nil, got %v", feederType, config.OptionalStringPtr)
			}
		} else {
			var expectedStr string
			switch v := val.(type) {
			case string:
				expectedStr = v
			case *string:
				if v == nil {
					if config.OptionalStringPtr != nil {
						t.Errorf("[%s] OptionalStringPtr: expected nil, got %v", feederType, config.OptionalStringPtr)
					}
					return
				}
				expectedStr = *v
			default:
				t.Errorf("[%s] OptionalStringPtr: unexpected type %T", feederType, val)
				return
			}
			if config.OptionalStringPtr == nil {
				t.Errorf("[%s] OptionalStringPtr: expected %q, got nil", feederType, expectedStr)
			} else if *config.OptionalStringPtr != expectedStr {
				t.Errorf("[%s] OptionalStringPtr: expected %q, got %q", feederType, expectedStr, *config.OptionalStringPtr)
			}
		}
	}

	if val, exists := expected["OptionalIntPtr"]; exists {
		if val == nil {
			if config.OptionalIntPtr != nil {
				t.Errorf("[%s] OptionalIntPtr: expected nil, got %v", feederType, config.OptionalIntPtr)
			}
		} else {
			var expectedInt int
			switch v := val.(type) {
			case int:
				expectedInt = v
			case *int:
				if v == nil {
					if config.OptionalIntPtr != nil {
						t.Errorf("[%s] OptionalIntPtr: expected nil, got %v", feederType, config.OptionalIntPtr)
					}
					return
				}
				expectedInt = *v
			default:
				t.Errorf("[%s] OptionalIntPtr: unexpected type %T", feederType, val)
				return
			}
			if config.OptionalIntPtr == nil {
				t.Errorf("[%s] OptionalIntPtr: expected %d, got nil", feederType, expectedInt)
			} else if *config.OptionalIntPtr != expectedInt {
				t.Errorf("[%s] OptionalIntPtr: expected %d, got %d", feederType, expectedInt, *config.OptionalIntPtr)
			}
		}
	}

	// Check slice field
	if val, exists := expected["OptionalSlice"]; exists {
		if val == nil {
			if config.OptionalSlice != nil {
				t.Errorf("[%s] OptionalSlice: expected nil, got %v", feederType, config.OptionalSlice)
			}
		} else {
			expectedSlice := val.([]string)
			if len(config.OptionalSlice) != len(expectedSlice) {
				t.Errorf("[%s] OptionalSlice: expected length %d, got length %d", feederType, len(expectedSlice), len(config.OptionalSlice))
			} else {
				for i, expected := range expectedSlice {
					if config.OptionalSlice[i] != expected {
						t.Errorf("[%s] OptionalSlice[%d]: expected %q, got %q", feederType, i, expected, config.OptionalSlice[i])
					}
				}
			}
		}
	}

	// Check nested struct field
	if val, exists := expected["OptionalNested"]; exists {
		if val == nil {
			if config.OptionalNested != nil {
				t.Errorf("[%s] OptionalNested: expected nil, got %v", feederType, config.OptionalNested)
			}
		} else {
			expectedNested := val.(*NestedConfig)
			if config.OptionalNested == nil {
				t.Errorf("[%s] OptionalNested: expected %+v, got nil", feederType, expectedNested)
			} else {
				if config.OptionalNested.Name != expectedNested.Name {
					t.Errorf("[%s] OptionalNested.Name: expected %q, got %q", feederType, expectedNested.Name, config.OptionalNested.Name)
				}
				if config.OptionalNested.Value != expectedNested.Value {
					t.Errorf("[%s] OptionalNested.Value: expected %d, got %d", feederType, expectedNested.Value, config.OptionalNested.Value)
				}
			}
		}
	}
}

// Test other tag modifiers besides omitempty
func TestTagModifiers_Comprehensive(t *testing.T) {
	type ConfigWithModifiers struct {
		// Different tag formats and modifiers
		FieldOmitempty    string `yaml:"field_omitempty,omitempty" json:"field_omitempty,omitempty" toml:"field_omitempty,omitempty"`
		FieldInline       string `yaml:",inline" json:",inline" toml:",inline"`
		FieldFlow         string `yaml:"field_flow,flow" json:"field_flow" toml:"field_flow"`
		FieldString       string `yaml:"field_string,string" json:"field_string,string" toml:"field_string"`
		FieldMultipleTags string `yaml:"field_multiple,omitempty,flow" json:"field_multiple,omitempty,string" toml:"field_multiple,omitempty"`
		FieldEmptyTagName string `yaml:",omitempty" json:",omitempty" toml:",omitempty"`
	}

	// Test with each feeder format
	testCases := []struct {
		name    string
		content string
		format  string
	}{
		{
			name: "yaml_with_modifiers",
			content: `
field_omitempty: "omitempty_value"
field_flow: "flow_value"
field_string: "string_value"
field_multiple: "multiple_value"
FieldEmptyTagName: "empty_tag_value"
`,
			format: "yaml",
		},
		{
			name: "json_with_modifiers",
			content: `{
  "field_omitempty": "omitempty_value",
  "field_flow": "flow_value", 
  "field_string": "string_value",
  "field_multiple": "multiple_value",
  "FieldEmptyTagName": "empty_tag_value"
}`,
			format: "json",
		},
		{
			name: "toml_with_modifiers",
			content: `
field_omitempty = "omitempty_value"
field_flow = "flow_value"
field_string = "string_value"
field_multiple = "multiple_value"
FieldEmptyTagName = "empty_tag_value"
`,
			format: "toml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp file
			tempFile, err := os.CreateTemp("", "test-modifiers-*."+tc.format)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			if _, err := tempFile.WriteString(tc.content); err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			tempFile.Close()

			var config ConfigWithModifiers
			var feeder interface{ Feed(interface{}) error }

			// Create appropriate feeder
			switch tc.format {
			case "yaml":
				feeder = NewYamlFeeder(tempFile.Name())
			case "json":
				feeder = NewJSONFeeder(tempFile.Name())
			case "toml":
				feeder = NewTomlFeeder(tempFile.Name())
			default:
				t.Fatalf("Unknown format: %s", tc.format)
			}

			err = feeder.Feed(&config)
			if err != nil {
				t.Fatalf("%s feeder failed: %v", tc.format, err)
			}

			// Verify that values are properly set despite tag modifiers
			if config.FieldOmitempty != "omitempty_value" {
				t.Errorf("[%s] FieldOmitempty: expected 'omitempty_value', got '%s'", tc.format, config.FieldOmitempty)
			}
			if config.FieldFlow != "flow_value" {
				t.Errorf("[%s] FieldFlow: expected 'flow_value', got '%s'", tc.format, config.FieldFlow)
			}
			if config.FieldString != "string_value" {
				t.Errorf("[%s] FieldString: expected 'string_value', got '%s'", tc.format, config.FieldString)
			}
			if config.FieldMultipleTags != "multiple_value" {
				t.Errorf("[%s] FieldMultipleTags: expected 'multiple_value', got '%s'", tc.format, config.FieldMultipleTags)
			}
			if config.FieldEmptyTagName != "empty_tag_value" {
				t.Errorf("[%s] FieldEmptyTagName: expected 'empty_tag_value', got '%s'", tc.format, config.FieldEmptyTagName)
			}
		})
	}
}

// Test standard library behavior for comparison
func TestStandardLibraryBehavior(t *testing.T) {
	type StandardConfig struct {
		RequiredField string `yaml:"required" json:"required" toml:"required"`
		OptionalField string `yaml:"optional,omitempty" json:"optional,omitempty" toml:"optional,omitempty"`
	}

	testData := map[string]string{
		"yaml": `
required: "test_value"
optional: "optional_value"
`,
		"json": `{
  "required": "test_value",
  "optional": "optional_value"
}`,
		"toml": `
required = "test_value"
optional = "optional_value"
`,
	}

	for format, content := range testData {
		t.Run("stdlib_"+format, func(t *testing.T) {
			var config StandardConfig

			switch format {
			case "yaml":
				err := yaml.Unmarshal([]byte(content), &config)
				if err != nil {
					t.Fatalf("YAML unmarshal failed: %v", err)
				}
			case "json":
				err := json.Unmarshal([]byte(content), &config)
				if err != nil {
					t.Fatalf("JSON unmarshal failed: %v", err)
				}
			case "toml":
				err := toml.Unmarshal([]byte(content), &config)
				if err != nil {
					t.Fatalf("TOML unmarshal failed: %v", err)
				}
			}

			// Standard libraries should handle omitempty correctly
			if config.RequiredField != "test_value" {
				t.Errorf("[%s] RequiredField: expected 'test_value', got '%s'", format, config.RequiredField)
			}
			if config.OptionalField != "optional_value" {
				t.Errorf("[%s] OptionalField: expected 'optional_value', got '%s'", format, config.OptionalField)
			}
		})
	}
}
