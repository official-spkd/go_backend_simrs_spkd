package patients

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	// Indonesian mobile numbers: 08xx..., 628xx... or +628xx..., 10-13 digits in total after the prefix.
	phonePattern = regexp.MustCompile(`^(\+62|62|0)8[1-9][0-9]{7,10}$`)

	// Names may contain letters (any script), spaces and . , ' -
	personNamePattern = regexp.MustCompile(`^\p{L}[\p{L} .,'\x{2019}-]*$`)
)

// RegisterValidators adds the custom validation tags used by this feature's DTOs.
func RegisterValidators() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return fmt.Errorf("unexpected validator engine")
	}

	rules := map[string]validator.Func{
		"phone":       func(fl validator.FieldLevel) bool { return phonePattern.MatchString(fl.Field().String()) },
		"person_name": func(fl validator.FieldLevel) bool { return personNamePattern.MatchString(fl.Field().String()) },
		"nik":         func(fl validator.FieldLevel) bool { return validNIK(fl.Field().String()) },
	}
	for tag, fn := range rules {
		if err := v.RegisterValidation(tag, fn); err != nil {
			return fmt.Errorf("register %q validator: %w", tag, err)
		}
	}
	return nil
}

// validNIK checks the structure of an Indonesian NIK: PPRRDD DDMMYY SSSS, where PP is the
// province code (11-94), DD is the birth day (+40 for women) and MM the birth month.
func validNIK(nik string) bool {
	if len(nik) != 16 {
		return false
	}
	part := func(from, to int) (int, bool) {
		n, err := strconv.Atoi(nik[from:to])
		return n, err == nil
	}

	province, ok := part(0, 2)
	if !ok || province < 11 || province > 94 {
		return false
	}
	day, ok := part(6, 8)
	if !ok {
		return false
	}
	if day > 40 {
		day -= 40
	}
	month, ok := part(8, 10)
	if !ok {
		return false
	}
	return day >= 1 && day <= 31 && month >= 1 && month <= 12
}
