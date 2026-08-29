package requestutil

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/go-playground/validator/v10"
)

var (
	ErrIncorrectFormatBody = errors.New("incorrect format body")
)

func HandlerRequest[T any](body io.ReadCloser, validate *validator.Validate) (*T, error) {
	var payload T
	errDecode := json.NewDecoder(body).Decode(&payload)
	if errDecode != nil {
		return nil, ErrIncorrectFormatBody
	}
	if errValidate := validate.Struct(&payload); errValidate != nil {
		return &payload, errValidate
	}
	return &payload, nil
}
