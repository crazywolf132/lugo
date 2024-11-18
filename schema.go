package lugo

import (
	"fmt"
	"reflect"
	"regexp"
)

// SchemaValidator defines validation rules for configuration
type SchemaValidator struct {
	// Required fields
	Required []string
	// Pattern matching for string fields
	Patterns map[string]*regexp.Regexp
	// Range validation for numeric fields
	Ranges map[string]struct {
		Min, Max float64
	}
	// Custom validation functions
	CustomValidators map[string]func(interface{}) error
	// Nested validators for complex types
	Nested map[string]*SchemaValidator
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		Patterns:         make(map[string]*regexp.Regexp),
		Ranges:           make(map[string]struct{ Min, Max float64 }),
		CustomValidators: make(map[string]func(interface{}) error),
		Nested:           make(map[string]*SchemaValidator),
	}
}

// AddPattern adds a regex pattern validation for a field
func (sv *SchemaValidator) AddPattern(field, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	sv.Patterns[field] = re
	return nil
}

// AddRange adds numeric range validation for a field
func (sv *SchemaValidator) AddRange(field string, min, max float64) {
	sv.Ranges[field] = struct{ Min, Max float64 }{min, max}
}

// AddCustomValidator adds a custom validation function for a field
func (sv *SchemaValidator) AddCustomValidator(field string, validator func(interface{}) error) {
	sv.CustomValidators[field] = validator
}

// AddNestedValidator adds a validator for nested structures
func (sv *SchemaValidator) AddNestedValidator(field string, validator *SchemaValidator) {
	sv.Nested[field] = validator
}

// Validate validates a configuration against the schema
func (sv *SchemaValidator) Validate(cfg interface{}) error {
	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", cfg)
	}

	typ := val.Type()

	// Check required fields
	for _, field := range sv.Required {
		f := val.FieldByName(field)
		if !f.IsValid() || f.IsZero() {
			return fmt.Errorf("required field %s is missing or empty", field)
		}
	}

	// Validate each field
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		fieldName := field.Name

		// Check patterns
		if re, ok := sv.Patterns[fieldName]; ok && fieldVal.Kind() == reflect.String {
			if !re.MatchString(fieldVal.String()) {
				return fmt.Errorf("field %s does not match pattern %s", fieldName, re)
			}
		}

		// Check ranges
		if r, ok := sv.Ranges[fieldName]; ok {
			var num float64
			switch fieldVal.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				num = float64(fieldVal.Int())
			case reflect.Float32, reflect.Float64:
				num = fieldVal.Float()
			}
			if num < r.Min || num > r.Max {
				return fmt.Errorf("field %s must be between %f and %f", fieldName, r.Min, r.Max)
			}
		}

		// Run custom validators
		if validator, ok := sv.CustomValidators[fieldName]; ok {
			if err := validator(fieldVal.Interface()); err != nil {
				return fmt.Errorf("field %s: %w", fieldName, err)
			}
		}

		// Check nested validators
		if nested, ok := sv.Nested[fieldName]; ok && fieldVal.Kind() == reflect.Struct {
			if err := nested.Validate(fieldVal.Interface()); err != nil {
				return fmt.Errorf("field %s: %w", fieldName, err)
			}
		}
	}

	return nil
}
