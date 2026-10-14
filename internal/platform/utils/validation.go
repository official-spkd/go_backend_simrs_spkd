package utils

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// InitValidation makes validation errors refer to JSON field names instead of Go field names.
func InitValidation() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	_ = v.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
		return strings.TrimSpace(fl.Field().String()) != ""
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
	for _, fe := range fieldErrs {
		msgs = append(msgs, fieldMessage(fe))
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
	case "email":
		return field + " must be a valid email address"
	case "numeric":
		return field + " must contain digits only"
	case "datetime":
		return field + " must use the format YYYY-MM-DD"
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
