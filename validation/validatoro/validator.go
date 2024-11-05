package validatoro

import "github.com/go-playground/validator/v10"

func New(options ...validator.Option) *validator.Validate {
	return validator.New(options...)
}
