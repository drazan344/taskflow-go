package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the go-playground validator with custom functionality
type Validator struct {
	validator *validator.Validate
}

// New creates a new validator instance
func New() *Validator {
	v := validator.New()
	
	// Register custom tag name function to use json tags
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	
	// Register custom validators
	registerCustomValidators(v)
	
	return &Validator{validator: v}
}

// ValidateStruct validates a struct and returns formatted error messages
func (v *Validator) ValidateStruct(s interface{}) []string {
	err := v.validator.Struct(s)
	if err == nil {
		return nil
	}

	var errors []string
	for _, err := range err.(validator.ValidationErrors) {
		errors = append(errors, formatValidationError(err))
	}
	
	return errors
}

// formatValidationError formats a validation error into a human-readable message
func formatValidationError(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()
	
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, err.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, err.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, err.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "password":
		return fmt.Sprintf("%s must contain at least one uppercase letter, one lowercase letter, and one number", field)
	case "priority":
		return fmt.Sprintf("%s must be one of: low, medium, high, critical", field)
	case "status":
		return fmt.Sprintf("%s must be a valid status", field)
	case "hexcolor":
		return fmt.Sprintf("%s must be a valid hex color (e.g., #FF5733)", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// registerCustomValidators registers custom validation rules
func registerCustomValidators(v *validator.Validate) {
	// Password validation - requires uppercase, lowercase, and number
	v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		if len(password) < 8 {
			return false
		}
		
		hasUpper := false
		hasLower := false
		hasNumber := false
		
		for _, char := range password {
			switch {
			case 'A' <= char && char <= 'Z':
				hasUpper = true
			case 'a' <= char && char <= 'z':
				hasLower = true
			case '0' <= char && char <= '9':
				hasNumber = true
			}
		}
		
		return hasUpper && hasLower && hasNumber
	})
	
	// Priority validation
	v.RegisterValidation("priority", func(fl validator.FieldLevel) bool {
		priority := fl.Field().String()
		validPriorities := []string{"low", "medium", "high", "critical"}
		for _, valid := range validPriorities {
			if priority == valid {
				return true
			}
		}
		return false
	})
	
	// Task status validation
	v.RegisterValidation("task_status", func(fl validator.FieldLevel) bool {
		status := fl.Field().String()
		validStatuses := []string{"todo", "in_progress", "review", "done"}
		for _, valid := range validStatuses {
			if status == valid {
				return true
			}
		}
		return false
	})
	
	// Project status validation
	v.RegisterValidation("project_status", func(fl validator.FieldLevel) bool {
		status := fl.Field().String()
		validStatuses := []string{"active", "paused", "completed", "archived"}
		for _, valid := range validStatuses {
			if status == valid {
				return true
			}
		}
		return false
	})
	
	// Hex color validation
	v.RegisterValidation("hexcolor", func(fl validator.FieldLevel) bool {
		color := fl.Field().String()
		if color == "" {
			return true // Optional field
		}
		matched, _ := regexp.MatchString(`^#(?:[0-9a-fA-F]{3}){1,2}$`, color)
		return matched
	})
}