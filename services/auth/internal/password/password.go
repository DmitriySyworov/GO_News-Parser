package password

import (
	"app/auth-service/internal/apperrors"
	"strings"

	"github.com/nbutton23/zxcvbn-go"
)

func ValidatePassword(password, email string, userInputs []string) error {
	if strings.Contains(password, email) {
		return apperrors.ErrPasswordContainEmail
	}
	res := zxcvbn.PasswordStrength(password, userInputs)
	if res.Score < 3 {
		return apperrors.ErrPasswordIsNotStrong
	}
	return nil
}
