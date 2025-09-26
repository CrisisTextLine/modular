package feeders

import (
	"os"
	"testing"
)

// OmitemptyTestConfig defines a structure with various omitempty tagged fields
type OmitemptyTestConfig struct {
	// Required fields (no omitempty)
	RequiredString string `yaml:"required_string" json:"required_string" toml:"required_string"`
	RequiredInt    int    `yaml:"required_int" json:"required_int" toml:"required_int"`

	// Optional fields with omitempty
	OptionalString  string  `yaml:"optional_string,omitempty" json:"optional_string,omitempty" toml:"optional_string,omitempty"`
	OptionalInt     int     `yaml:"optional_int,omitempty" json:"optional_int,omitempty" toml:"optional_int,omitempty"`
	OptionalBool    bool    `yaml:"optional_bool,omitempty" json:"optional_bool,omitempty" toml:"optional_bool,omitempty"`
	OptionalFloat64 float64 `yaml:"optional_float64,omitempty" json:"optional_float64,omitempty" toml:"optional_float64,omitempty"`

	// Pointer fields with omitempty
	OptionalStringPtr *string `yaml:"optional_string_ptr,omitempty" json:"optional_string_ptr,omitempty" toml:"optional_string_ptr,omitempty"`
	OptionalIntPtr    *int    `yaml:"optional_int_ptr,omitempty" json:"optional_int_ptr,omitempty" toml:"optional_int_ptr,omitempty"`

	// Slice fields with omitempty
	OptionalSlice []string `yaml:"optional_slice,omitempty" json:"optional_slice,omitempty" toml:"optional_slice,omitempty"`

	// Nested struct with omitempty
	OptionalNested *NestedConfig `yaml:"optional_nested,omitempty" json:"optional_nested,omitempty" toml:"optional_nested,omitempty"`
}

type NestedConfig struct {
	Name  string `yaml:"name" json:"name" toml:"name"`
	Value int    `yaml:"value" json:"value" toml:"value"`
}

func TestYAMLFeeder_OmitemptyHandling(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		expectFields map[string]interface{}
	}{
		{
			name: "all_fields_present",
			yamlContent: `
required_string: "test_string"
required_int: 42
optional_string: "optional_value"
optional_int: 123
optional_bool: true
optional_float64: 3.14
optional_string_ptr: "pointer_value"
optional_int_ptr: 456
optional_slice:
  - "item1"
  - "item2"
optional_nested:
  name: "nested_name"
  value: 789
`,
			expectFields: map[string]interface{}{
				"RequiredString":    "test_string",
				"RequiredInt":       42,
				"OptionalString":    "optional_value",
				"OptionalInt":       123,
				"OptionalBool":      true,
				"OptionalFloat64":   3.14,
				"OptionalStringPtr": "pointer_value",
				"OptionalIntPtr":    456,
				"OptionalSlice":     []string{"item1", "item2"},
				"OptionalNested":    &NestedConfig{Name: "nested_name", Value: 789},
			},
		},
		{
			name: "only_required_fields",
			yamlContent: `
required_string: "required_only"
required_int: 999
`,
			expectFields: map[string]interface{}{
				"RequiredString": "required_only",
				"RequiredInt":    999,
				// Optional fields should have zero values
				"OptionalString":    "",
				"OptionalInt":       0,
				"OptionalBool":      false,
				"OptionalFloat64":   0.0,
				"OptionalStringPtr": (*string)(nil),
				"OptionalIntPtr":    (*int)(nil),
				"OptionalSlice":     ([]string)(nil),
				"OptionalNested":    (*NestedConfig)(nil),
			},
		},
		{
			name: "mixed_fields",
			yamlContent: `
required_string: "mixed_test"
required_int: 555
optional_string: "has_value"
optional_int: 777
# optional_bool is not provided
# optional_float64 is not provided
optional_string_ptr: "ptr_value"
# optional_int_ptr is not provided
optional_slice:
  - "single_item"
# optional_nested is not provided
`,
			expectFields: map[string]interface{}{
				"RequiredString":    "mixed_test",
				"RequiredInt":       555,
				"OptionalString":    "has_value",
				"OptionalInt":       777,
				"OptionalBool":      false, // zero value
				"OptionalFloat64":   0.0,   // zero value
				"OptionalStringPtr": "ptr_value",
				"OptionalIntPtr":    (*int)(nil),
				"OptionalSlice":     []string{"single_item"},
				"OptionalNested":    (*NestedConfig)(nil),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp YAML file
			tempFile, err := os.CreateTemp("", "test-omitempty-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			if _, err := tempFile.WriteString(tt.yamlContent); err != nil {
				t.Fatalf("Failed to write YAML content: %v", err)
			}
			tempFile.Close()

			// Test YAML feeder
			feeder := NewYamlFeeder(tempFile.Name())
			var config OmitemptyTestConfig

			err = feeder.Feed(&config)
			if err != nil {
				t.Fatalf("YAML feeder failed: %v", err)
			}

			// Verify expected fields
			verifyOmitemptyTestConfig(t, "YAML", &config, tt.expectFields)
		})
	}
}
