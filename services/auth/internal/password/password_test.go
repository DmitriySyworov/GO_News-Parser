package password_test

import (
	"app/auth-service/internal/password"
	"testing"
)

var ValidatePasswordDataCaseSuccess = []struct {
	TestName string
	Name     string
	Email    string
	Password string
}{
	{
		TestName: "standard valid password - ",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "Qwashksa&8S*(jss",
	},
	{
		TestName: "password with spaces inside - ",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "Correct Horse Battery Staple 2026!",
	},
	{
		TestName: "minimum allowed length - ",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "z*1!#sDf",
	}, {
		TestName: "name/email as part of words - ",
		Name:     "Bob",
		Email:    "bob@gmail.com",
		Password: "Light**Building**991!",
	},
}

func TestValidatePasswordSuccess(t *testing.T) {
	for _, test := range ValidatePasswordDataCaseSuccess {
		t.Run(test.TestName, func(t *testing.T) {
			errValidatePassword := password.ValidatePassword(test.Password, test.Email, []string{test.Name})
			if errValidatePassword != nil {
				t.Fatal(errValidatePassword)
			}
		})
	}
}

var ValidatePasswordDataCaseNegative = []struct {
	TestName string
	Name     string
	Email    string
	Password string
}{
	{
		TestName: "password contains email - ",
		Name:     "Bob", Email: "example@gmail.com",
		Password: "Qwansiexample@gmail.comjz!@8sUSao",
	},
	{
		TestName: "password contains name - ",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "dfgBobaaaaaaa",
	},
	{
		TestName: "password contains name in lowercase - ",
		Name:     "BOB",
		Email:    "example@gmail.com",
		Password: "super_bob_password123!",
	},
	{
		TestName: "password is too short",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "Short1!",
	},
	{
		TestName: "password is empty",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "",
	},

	{
		TestName: "password without digits",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "NoDigitsPassword!",
	},
	{
		TestName: "password without special symbols",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "NoSpecialSymbols123",
	},

	{
		TestName: "password is simple/popular",
		Name:     "Bob", Email: "example@gmail.com",
		Password: "qwerty123",
	},
	{
		TestName: "password with repeating sequences",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "11111111Aaaa!",
	},

	{
		TestName: "password consists of spaces",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "            ",
	},
	{
		TestName: "password with trailing spaces only",
		Name:     "Bob",
		Email:    "example@gmail.com",
		Password: "Password123!   ",
	},
}

func TestValidatePasswordNegative(t *testing.T) {
	for _, test := range ValidatePasswordDataCaseNegative {
		t.Run(test.TestName, func(t *testing.T) {
			errValidatePassword := password.ValidatePassword(test.Password, test.Email, []string{test.TestName})
			if errValidatePassword == nil {
				t.Fatal(errValidatePassword)
			}
		})
	}
}
