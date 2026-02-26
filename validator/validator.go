package validator

import (
	"fmt"
	"strings"

	"github.com/alimzhanovlr/sdk/errors"
	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground validator
type Validator struct {
	validate *validator.Validate
}

// New creates a new validator instance
func New() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

// Validate validates a struct
func (v *Validator) Validate(data interface{}) error {
	if err := v.validate.Struct(data); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// formatValidationError formats validation errors into AppError
func (v *Validator) formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		details := make(map[string]interface{})

		for _, e := range validationErrors {
			field := strings.ToLower(e.Field())
			details[field] = formatFieldError(e)
		}

		return errors.ErrValidation.WithDetails(details)
	}

	return errors.Wrap(err, "validation_error", "Validation failed", 400)
}

// formatFieldError formats a single field validation error
func formatFieldError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", e.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", e.Field(), e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", e.Field(), e.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", e.Field(), e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", e.Field(), e.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", e.Field(), e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", e.Field(), e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", e.Field(), e.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", e.Field())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", e.Field())
	default:
		return fmt.Sprintf("%s failed on %s validation", e.Field(), e.Tag())
	}
}

// RegisterCustomValidation registers a custom validation function
func (v *Validator) RegisterCustomValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}
