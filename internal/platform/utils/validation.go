package utils

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// License numbers (SIP, STR) mix digits, letters and the separators . / -
var licensePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9./-]*$`)

// InitValidation makes validation errors refer to JSON field names instead of Go field names.
func InitValidation() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	_ = v.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
		return strings.TrimSpace(fl.Field().String()) != ""
	})
	_ = v.RegisterValidation("license_number", func(fl validator.FieldLevel) bool {
		return licensePattern.MatchString(fl.Field().String())
	})
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
}

// BindQuery binds and validates query parameters. On failure it writes a 400 response and
// returns false.
func BindQuery(c *gin.Context, dst any) bool {
	if err := c.ShouldBindQuery(dst); err != nil {
		AbortWithError(c, http.StatusBadRequest, validationMessage(err))
		return false
	}
	return true
}

// BindJSON decodes the request body into dst and validates it. On failure it writes a 400
// response with a readable message and returns false.
func BindJSON(c *gin.Context, dst any) bool {
	err := c.ShouldBindJSON(dst)
	if err == nil {
		return true
	}
	AbortWithError(c, http.StatusBadRequest, validationMessage(err))
	return false
}

func validationMessage(err error) string {
	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return "invalid request body"
	}

	msgs := make([]string, 0, len(fieldErrs))
	seen := make(map[string]bool, len(fieldErrs))
	for _, fe := range fieldErrs {
		msg := fieldMessage(fe)
		if !seen[msg] {
			seen[msg] = true
			msgs = append(msgs, msg)
		}
	}
	return strings.Join(msgs, "; ")
}

func fieldMessage(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(fe.Param(), " ", ", "))
	case "uuid":
		return field + " must be a valid UUID"
	case "email":
		return field + " must be a valid email address"
	case "numeric":
		return field + " must contain digits only"
	case "datetime":
		return field + " must use the format YYYY-MM-DD"
	case "required_if":
		return field + " is required when " + condition(fe.Param())
	case "excluded_unless":
		return field + " is only allowed when " + condition(fe.Param())
	case "bpjs_id":
		return field + " must be a 13-digit BPJS number or a 16-digit NIK"
	case "license_number":
		return field + " may only contain letters, digits and . / -"
	case "alpha":
		return field + " may only contain letters"
	case "lte":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "notblank":
		return field + " must not be blank"
	case "person_name":
		return field + " may only contain letters, spaces and . , ' -"
	case "nik":
		return field + " is not a valid NIK"
	case "phone":
		return field + " must be a valid Indonesian mobile number (e.g. 081234567890)"
	default:
		return field + " is invalid"
	}
}

// condition turns a "Field value" validator parameter into "field is value".
func condition(param string) string {
	name, value, _ := strings.Cut(param, " ")
	return toSnake(name) + " is " + value
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 && unicode.IsLower(rune(s[i-1])) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
