package validator

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// New returns a configured validator instance.
// Using go-playground/validator is part of the required stack.
func New() *validator.Validate {
	v := validator.New(validator.WithPrivateFieldValidation())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

var (
	once sync.Once
	v    *validator.Validate
)

// Default returns a singleton validator instance.
func Default() *validator.Validate {
	once.Do(func() { v = New() })
	return v
}
