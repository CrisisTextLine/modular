package jsonschema

import (
	"fmt"
	"strings"
)

// JSON validation operation step methods

func (ctx *JSONSchemaBDDTestContext) iValidateValidUserJSONData() error {
	if ctx.service == nil || ctx.compiledSchema == nil {
		return fmt.Errorf("jsonschema service or schema not available")
	}

	validJSON := []byte(`{"name": "John Doe", "age": 30}`)

	err := ctx.service.ValidateBytes(ctx.compiledSchema, validJSON)
	if err != nil {
		ctx.lastError = err
		ctx.validationPass = false
	} else {
		ctx.validationPass = true
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) theValidationShouldPass() error {
	if !ctx.validationPass {
		return fmt.Errorf("validation should have passed but failed: %v", ctx.lastError)
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateInvalidUserJSONData() error {
	if ctx.service == nil || ctx.compiledSchema == nil {
		return fmt.Errorf("jsonschema service or schema not available")
	}

	invalidJSON := []byte(`{"age": "not a number"}`) // Missing required "name" field, invalid type for age

	err := ctx.service.ValidateBytes(ctx.compiledSchema, invalidJSON)
	if err != nil {
		ctx.lastError = err
		ctx.validationPass = false
	} else {
		ctx.validationPass = true
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) theValidationShouldFailWithAppropriateErrors() error {
	if ctx.validationPass {
		return fmt.Errorf("validation should have failed but passed")
	}

	if ctx.lastError == nil {
		return fmt.Errorf("expected validation error but got none")
	}

	// Check that error message contains useful information
	errMsg := ctx.lastError.Error()
	if errMsg == "" {
		return fmt.Errorf("validation error message is empty")
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateDataFromBytes() error {
	if ctx.service == nil || ctx.compiledSchema == nil {
		return fmt.Errorf("jsonschema service or schema not available")
	}

	testData := []byte(`{"name": "Test User", "age": 25}`)
	err := ctx.service.ValidateBytes(ctx.compiledSchema, testData)
	if err != nil {
		ctx.lastError = err
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateDataFromReader() error {
	if ctx.service == nil || ctx.compiledSchema == nil {
		return fmt.Errorf("jsonschema service or schema not available")
	}

	testData := `{"name": "Test User", "age": 25}`
	reader := strings.NewReader(testData)

	err := ctx.service.ValidateReader(ctx.compiledSchema, reader)
	if err != nil {
		ctx.lastError = err
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateDataFromInterface() error {
	if ctx.service == nil || ctx.compiledSchema == nil {
		return fmt.Errorf("jsonschema service or schema not available")
	}

	testData := map[string]interface{}{
		"name": "Test User",
		"age":  25,
	}

	err := ctx.service.ValidateInterface(ctx.compiledSchema, testData)
	if err != nil {
		ctx.lastError = err
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) allValidationMethodsShouldWorkCorrectly() error {
	if ctx.lastError != nil {
		return fmt.Errorf("one or more validation methods failed: %v", ctx.lastError)
	}

	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateValidUserJSONDataWithBytesMethod() error {
	if ctx.compiledSchema == nil {
		// Create a user data schema first
		if err := ctx.iHaveACompiledSchemaForUserData(); err != nil {
			return err
		}
	}

	validJSON := `{"name": "John Doe", "age": 30}`
	ctx.lastError = ctx.service.ValidateBytes(ctx.compiledSchema, []byte(validJSON))
	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateInvalidUserJSONDataWithBytesMethod() error {
	if ctx.compiledSchema == nil {
		// Create a user data schema first
		if err := ctx.iHaveACompiledSchemaForUserData(); err != nil {
			return err
		}
	}

	invalidJSON := `{"age": "not a number"}` // missing required "name" field and age is not a number
	ctx.lastError = ctx.service.ValidateBytes(ctx.compiledSchema, []byte(invalidJSON))
	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateDataUsingTheReaderMethod() error {
	if ctx.compiledSchema == nil {
		// Create a user data schema first
		if err := ctx.iHaveACompiledSchemaForUserData(); err != nil {
			return err
		}
	}

	validJSON := `{"name": "John Doe", "age": 30}`
	reader := strings.NewReader(validJSON)
	ctx.lastError = ctx.service.ValidateReader(ctx.compiledSchema, reader)
	return nil
}

func (ctx *JSONSchemaBDDTestContext) iValidateDataUsingTheInterfaceMethod() error {
	if ctx.compiledSchema == nil {
		// Create a user data schema first
		if err := ctx.iHaveACompiledSchemaForUserData(); err != nil {
			return err
		}
	}

	userData := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	}
	ctx.lastError = ctx.service.ValidateInterface(ctx.compiledSchema, userData)
	return nil
}
