package feeders

import (
	"os"
	"testing"
)

func TestTOMLFeeder_OmitemptyHandling(t *testing.T) {
	tests := []struct {
		name         string
		tomlContent  string
		expectFields map[string]interface{}
	}{
		{
			name: "all_fields_present",
			tomlContent: `
required_string = "test_string"
required_int = 42
optional_string = "optional_value"
optional_int = 123
optional_bool = true
optional_float64 = 3.14
optional_string_ptr = "pointer_value"
optional_int_ptr = 456
optional_slice = ["item1", "item2"]

[optional_nested]
name = "nested_name"
value = 789
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
			tomlContent: `
required_string = "required_only"
required_int = 999
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
			tomlContent: `
required_string = "mixed_test"
required_int = 555
optional_string = "has_value"
optional_int = 777
optional_string_ptr = "ptr_value"
optional_slice = ["single_item"]
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
			// Create temp TOML file
			tempFile, err := os.CreateTemp("", "test-omitempty-*.toml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			if _, err := tempFile.WriteString(tt.tomlContent); err != nil {
				t.Fatalf("Failed to write TOML content: %v", err)
			}
			tempFile.Close()

			// Test TOML feeder
			feeder := NewTomlFeeder(tempFile.Name())
			var config OmitemptyTestConfig

			err = feeder.Feed(&config)
			if err != nil {
				t.Fatalf("TOML feeder failed: %v", err)
			}

			// Verify expected fields
			verifyOmitemptyTestConfig(t, "TOML", &config, tt.expectFields)
		})
	}
}

func TestJSONFeeder_OmitemptyHandling(t *testing.T) {
	tests := []struct {
		name         string
		jsonContent  string
		expectFields map[string]interface{}
	}{
		{
			name: "all_fields_present",
			jsonContent: `{
  "required_string": "test_string",
  "required_int": 42,
  "optional_string": "optional_value",
  "optional_int": 123,
  "optional_bool": true,
  "optional_float64": 3.14,
  "optional_string_ptr": "pointer_value",
  "optional_int_ptr": 456,
  "optional_slice": ["item1", "item2"],
  "optional_nested": {
    "name": "nested_name",
    "value": 789
  }
}`,
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
			jsonContent: `{
  "required_string": "required_only",
  "required_int": 999
}`,
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
			jsonContent: `{
  "required_string": "mixed_test",
  "required_int": 555,
  "optional_string": "has_value",
  "optional_int": 777,
  "optional_string_ptr": "ptr_value",
  "optional_slice": ["single_item"]
}`,
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
		{
			name: "null_values_in_json",
			jsonContent: `{
  "required_string": "null_test",
  "required_int": 111,
  "optional_string": "has_value",
  "optional_string_ptr": null,
  "optional_int_ptr": null,
  "optional_nested": null
}`,
			expectFields: map[string]interface{}{
				"RequiredString":    "null_test",
				"RequiredInt":       111,
				"OptionalString":    "has_value",
				"OptionalInt":       0,     // zero value
				"OptionalBool":      false, // zero value
				"OptionalFloat64":   0.0,   // zero value
				"OptionalStringPtr": (*string)(nil),
				"OptionalIntPtr":    (*int)(nil),
				"OptionalSlice":     ([]string)(nil),
				"OptionalNested":    (*NestedConfig)(nil),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp JSON file
			tempFile, err := os.CreateTemp("", "test-omitempty-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			if _, err := tempFile.WriteString(tt.jsonContent); err != nil {
				t.Fatalf("Failed to write JSON content: %v", err)
			}
			tempFile.Close()

			// Test JSON feeder
			feeder := NewJSONFeeder(tempFile.Name())
			var config OmitemptyTestConfig

			err = feeder.Feed(&config)
			if err != nil {
				t.Fatalf("JSON feeder failed: %v", err)
			}

			// Verify expected fields
			verifyOmitemptyTestConfig(t, "JSON", &config, tt.expectFields)
		})
	}
}
