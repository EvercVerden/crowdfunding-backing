package util

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// ValidateFutureDate 验证日期是否在未来
func ValidateFutureDate(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	return date.After(time.Now())
}
