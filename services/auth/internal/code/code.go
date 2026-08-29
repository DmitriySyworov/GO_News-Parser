package code

import (
	"crypto/rand"
	"math/big"
)

func GenerateCode() (int, error) {
	maxValue := big.NewInt(900000)
	code, errRand := rand.Int(rand.Reader, maxValue)
	if errRand != nil {
		return 0, errRand
	}
	return int(code.Int64()) + 100000, nil
}
