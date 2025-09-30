package jsonschema

import (
	"fmt"
	"os"
)

// Schema compilation step methods

func (ctx *JSONSchemaBDDTestContext) iCompileASchemaFromAJSONString() error {
	if ctx.service == nil {
		return fmt.Errorf("jsonschema service not available")
	}

	// Create a temporary schema file
	schemaString := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"]
	}`

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "schema-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(schemaString)
	if err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	ctx.tempFile = tmpFile.Name()

	schema, err := ctx.service.CompileSchema(ctx.tempFile)
	if err != nil {
		ctx.lastError = err
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	ctx.compiledSchema = schema
	return nil
}

func (ctx *JSONSchemaBDDTestContext) theSchemaShouldBeCompiledSuccessfully() error {
	if ctx.compiledSchema == nil {
		return fmt.Errorf("schema was not compiled")
	}

	if ctx.lastError != nil {
		return fmt.Errorf("schema compilation failed: %v", ctx.lastError)
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) iHaveACompiledSchemaForUserData() error {
	return ctx.iCompileASchemaFromAJSONString()
}

func (ctx *JSONSchemaBDDTestContext) iHaveACompiledSchema() error {
	return ctx.iCompileASchemaFromAJSONString()
}

func (ctx *JSONSchemaBDDTestContext) iTryToCompileAnInvalidSchema() error {
	if ctx.service == nil {
		return fmt.Errorf("jsonschema service not available")
	}

	invalidSchemaString := `{"type": "invalid_type"}` // Invalid schema type

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "invalid-schema-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(invalidSchemaString)
	if err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	_, err = ctx.service.CompileSchema(tmpFile.Name())
	if err != nil {
		ctx.lastError = err
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) aSchemaCompilationErrorShouldBeReturned() error {
	if ctx.lastError == nil {
		return fmt.Errorf("expected schema compilation error but got none")
	}

	// Check that error message contains useful information
	errMsg := ctx.lastError.Error()
	if errMsg == "" {
		return fmt.Errorf("schema compilation error message is empty")
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) iCompileAValidSchema() error {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "number"}
		},
		"required": ["name"]
	}`

	// Create temporary file for schema
	tempFile, err := os.CreateTemp("", "test-schema-*.json")
	if err != nil {
		return err
	}
	defer tempFile.Close()

	ctx.tempFile = tempFile.Name()
	if _, err := tempFile.WriteString(schemaJSON); err != nil {
		return err
	}

	// Compile the schema
	ctx.compiledSchema, ctx.lastError = ctx.service.CompileSchema(ctx.tempFile)
	return nil
}
