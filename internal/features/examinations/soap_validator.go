package examinations

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// ICD-10 codes look like J06 or J06.9 (letter, two digits, optional dot and up to four characters).
var icd10Pattern = regexp.MustCompile(`^[A-Za-z][0-9]{2}(\.[0-9A-Za-z]{1,4})?$`)

// RegisterValidators adds the custom validation tags used by this feature's DTOs.
func RegisterValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return fmt.Errorf("unexpected validator engine")
	}
	err := v.RegisterValidation("icd10", func(fl validator.FieldLevel) bool {
		return icd10Pattern.MatchString(fl.Field().String())
	})
	if err != nil {
		return fmt.Errorf("register icd10 validator: %w", err)
	}
	return nil
}
