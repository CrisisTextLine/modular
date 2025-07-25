package feeders

import (
	"errors"
	"fmt"
)

// Static error definitions for feeders to comply with linting rules

// DotEnv feeder errors
var (
	ErrDotEnvInvalidStructureType = errors.New("expected pointer to struct")
	ErrDotEnvUnsupportedType      = errors.New("unsupported type")
	ErrDotEnvInvalidLineFormat    = errors.New("invalid .env line format")
)

// JSON feeder errors
var (
	ErrJSONExpectedMapForStruct      = errors.New("expected map for struct field")
	ErrJSONCannotConvert             = errors.New("cannot convert value to field type")
	ErrJSONCannotConvertSliceElement = errors.New("cannot convert slice element")
	ErrJSONExpectedArrayForSlice     = errors.New("expected array for slice field")
	ErrJSONFieldCannotBeSet          = errors.New("field cannot be set")
	ErrJSONArraySizeExceeded         = errors.New("array size exceeded")
)

// TOML feeder errors
var (
	ErrTomlExpectedMapForStruct      = errors.New("expected map for struct field")
	ErrTomlCannotConvert             = errors.New("cannot convert value to field type")
	ErrTomlCannotConvertSliceElement = errors.New("cannot convert slice element")
	ErrTomlExpectedArrayForSlice     = errors.New("expected array for slice field")
	ErrTomlFieldCannotBeSet          = errors.New("field cannot be set")
	ErrTomlArraySizeExceeded         = errors.New("array size exceeded")
)

// YAML feeder errors
var (
	ErrYamlFieldCannotBeSet     = errors.New("field cannot be set")
	ErrYamlUnsupportedFieldType = errors.New("unsupported field type")
	ErrYamlTypeConversion       = errors.New("type conversion error")
	ErrYamlBoolConversion       = errors.New("cannot convert string to bool")
	ErrYamlExpectedMap          = errors.New("expected map for field")
	ErrYamlExpectedArray        = errors.New("expected array for field")
	ErrYamlArraySizeExceeded    = errors.New("array size exceeded")
	ErrYamlExpectedMapForSlice  = errors.New("expected map for slice element")
)

// General feeder errors
var (
	ErrJsonFeederUnavailable = errors.New("json feeder unavailable")
	ErrTomlFeederUnavailable = errors.New("toml feeder unavailable")
)

// Helper functions to create wrapped errors with context
func wrapDotEnvStructureError(got interface{}) error {
	return fmt.Errorf("%w, got %T", ErrDotEnvInvalidStructureType, got)
}

func wrapDotEnvUnsupportedTypeError(typeName string) error {
	return fmt.Errorf("%w: %s", ErrDotEnvUnsupportedType, typeName)
}

func wrapJSONMapError(fieldPath string, got interface{}) error {
	return fmt.Errorf("%w %s, got %T", ErrJSONExpectedMapForStruct, fieldPath, got)
}

func wrapJSONConvertError(value interface{}, fieldType, fieldPath string) error {
	return fmt.Errorf("%w %T to %s for field %s", ErrJSONCannotConvert, value, fieldType, fieldPath)
}

func wrapJSONSliceElementError(item interface{}, elemType, fieldPath string, index int) error {
	return fmt.Errorf("%w %T to %s for field %s[%d]", ErrJSONCannotConvertSliceElement, item, elemType, fieldPath, index)
}

func wrapJSONArrayError(fieldPath string, got interface{}) error {
	return fmt.Errorf("%w %s, got %T", ErrJSONExpectedArrayForSlice, fieldPath, got)
}

func wrapTomlMapError(fieldPath string, got interface{}) error {
	return fmt.Errorf("%w %s, got %T", ErrTomlExpectedMapForStruct, fieldPath, got)
}

func wrapTomlConvertError(value interface{}, fieldType, fieldPath string) error {
	return fmt.Errorf("%w %T to %s for field %s", ErrTomlCannotConvert, value, fieldType, fieldPath)
}

func wrapTomlSliceElementError(item interface{}, elemType, fieldPath string, index int) error {
	return fmt.Errorf("%w %T to %s for field %s[%d]", ErrTomlCannotConvertSliceElement, item, elemType, fieldPath, index)
}

func wrapTomlArrayError(fieldPath string, got interface{}) error {
	return fmt.Errorf("%w %s, got %T", ErrTomlExpectedArrayForSlice, fieldPath, got)
}

// YAML error wrapper functions
func wrapYamlFieldCannotBeSetError() error {
	return fmt.Errorf("%w", ErrYamlFieldCannotBeSet)
}

func wrapYamlUnsupportedFieldTypeError(fieldType string) error {
	return fmt.Errorf("%w: %s", ErrYamlUnsupportedFieldType, fieldType)
}

func wrapYamlTypeConversionError(fromType, toType string) error {
	return fmt.Errorf("%w: cannot convert %s to %s", ErrYamlTypeConversion, fromType, toType)
}

func wrapYamlBoolConversionError(value string) error {
	return fmt.Errorf("%w: '%s'", ErrYamlBoolConversion, value)
}

// Wrapper functions for JSON feeder errors
func wrapJSONFieldCannotBeSet(fieldPath string) error {
	return fmt.Errorf("%w: %s", ErrJSONFieldCannotBeSet, fieldPath)
}

// Wrapper functions for TOML feeder errors
func wrapTomlFieldCannotBeSet(fieldPath string) error {
	return fmt.Errorf("%w: %s", ErrTomlFieldCannotBeSet, fieldPath)
}

func wrapTomlArraySizeExceeded(fieldPath string, arraySize, maxSize int) error {
	return fmt.Errorf("%w: array %s has %d elements but field can only hold %d", ErrTomlArraySizeExceeded, fieldPath, arraySize, maxSize)
}

func wrapJSONArraySizeExceeded(fieldPath string, arraySize, maxSize int) error {
	return fmt.Errorf("%w: array %s has %d elements but field can only hold %d", ErrJSONArraySizeExceeded, fieldPath, arraySize, maxSize)
}

// Additional YAML error wrapper functions
func wrapYamlExpectedMapError(fieldPath string, got interface{}) error {
	return fmt.Errorf("%w %s, got %T", ErrYamlExpectedMap, fieldPath, got)
}

func wrapYamlExpectedArrayError(fieldPath string, got interface{}) error {
	return fmt.Errorf("%w %s, got %T", ErrYamlExpectedArray, fieldPath, got)
}

func wrapYamlArraySizeExceeded(fieldPath string, arraySize, maxSize int) error {
	return fmt.Errorf("%w: array %s has %d elements but field can only hold %d", ErrYamlArraySizeExceeded, fieldPath, arraySize, maxSize)
}

func wrapYamlExpectedMapForSliceError(fieldPath string, index int, got interface{}) error {
	return fmt.Errorf("%w %d in field %s, got %T", ErrYamlExpectedMapForSlice, index, fieldPath, got)
}
