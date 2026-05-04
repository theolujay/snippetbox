package validator

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// Contains a map of validation errors for form fields
type Validator struct {
	FieldErrors map[string]string
}

// Valid() returns true if the FieldErrors map doesn't
// contain any entries
func (v *Validator) Valid() bool {
	return len(v.FieldErrors) == 0
}

// AddFieldError() adds an error message to the FieldErrors map
// (so long as no entry already exists or the given key)
func (v *Validator) AddFieldError(key, message string) {
	// Initialize the map if necessary
	if v.FieldErrors == nil {
		v.FieldErrors = map[string]string{}
	}
	// Only add field error key doesn't already exist
	if _, exists := v.FieldErrors[key]; !exists {
		v.FieldErrors[key] = message
	}
}

// CheckField() checks if a field is not OK and adds the error
func (v *Validator) Checkfield(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}

// NotBlank() returns true if the value is not an empty string
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MaxChars() returns true if a value contains no more than n characters
func MaxChars(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// PermittedInt() returns true if a value is in a list of permitted integers
func PermittedInt(value int, permittedValues ...int) bool {
	return slices.Contains(permittedValues, value)
}
