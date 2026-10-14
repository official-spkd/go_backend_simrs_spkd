package visits

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RegisterValidators adds the custom validation tags used by this feature's DTOs.
func RegisterValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return fmt.Errorf("unexpected validator engine")
	}
	if err := v.RegisterValidation("bpjs_id", func(fl validator.FieldLevel) bool {
		return validBPJSID(fl.Field().String())
	}); err != nil {
		return fmt.Errorf("register bpjs_id validator: %w", err)
	}
	return nil
}

// validBPJSID accepts a 13-digit BPJS card number or a 16-digit NIK.
func validBPJSID(s string) bool {
	if len(s) != 13 && len(s) != 16 {
		return false
	}
	return strings.Trim(s, "0123456789") == ""
}
