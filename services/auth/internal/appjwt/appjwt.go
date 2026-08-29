package appjwt

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/common"
	"errors"
	"shared/loggers"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWT struct {
	Logger    *loggers.Logger
	Signature []byte
}

func NewJWT(signature string, logger *loggers.Logger) *JWT {
	return &JWT{
		Signature: []byte(signature),
		Logger:    logger,
	}
}
func (j *JWT) CreateSessionJWT(sessionID string) (string, error) {
	claim := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type": "session",
		"sub":  sessionID,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Minute * 5).Unix(),
	})
	token, errToken := claim.SignedString(j.Signature)
	if errToken != nil {
		j.Logger.Error("failed to sign token: ", errToken)
		return "", nil
	}
	return token, nil
}
func (j *JWT) CreateAccessJWT(userUUID string) (string, error) {
	claim := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type": "access",
		"sub":  userUUID,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(common.TTLAccessKey).Unix(),
	})
	token, errToken := claim.SignedString(j.Signature)
	if errToken != nil {
		j.Logger.Error("failed to sign token: ", errToken)
		return "", nil
	}
	return token, nil
}
func (j *JWT) CreateRefreshJWT(userUUID, refreshUUID string) (string, error) {
	claim := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type": "refresh",
		"sub":  userUUID,
		"jti":  refreshUUID,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(common.TTLRefreshKey).Unix(),
	})
	token, errToken := claim.SignedString(j.Signature)
	if errToken != nil {
		j.Logger.Error("failed to sign token: ", errToken)
		return "", nil
	}
	return token, nil
}

var (
	ErrExpiredRefreshToken   = errors.New("refresh token has expired")
	ErrIncorrectRefreshToken = errors.New("incorrect refresh token")
)

type RefreshToken struct {
	UserUUID    string
	RefreshUUID string
}

func (j *JWT) ParseRefreshToken(refreshToken string) (*RefreshToken, error) {
	token, errToken := jwt.Parse(refreshToken, func(token *jwt.Token) (any, error) {
		return j.Signature, nil
	})
	if errToken != nil {
		if errors.Is(errToken, jwt.ErrTokenExpired) {
			return nil, ErrExpiredRefreshToken
		}
		return nil, ErrIncorrectRefreshToken
	}
	if types, okType := token.Claims.(jwt.MapClaims)["type"].(string); !okType || types != "refresh" {
		return nil, ErrIncorrectRefreshToken
	}
	userUUID, okUserUUID := token.Claims.(jwt.MapClaims)["sub"].(string)
	if !okUserUUID {
		return nil, ErrIncorrectRefreshToken
	}
	refreshUUID, okRefreshUUID := token.Claims.(jwt.MapClaims)["jti"].(string)
	if !okRefreshUUID {
		return nil, ErrIncorrectRefreshToken
	}
	return &RefreshToken{
		UserUUID:    userUUID,
		RefreshUUID: refreshUUID,
	}, nil

}
func (j *JWT) ParseSessionToken(accessToken string) (string, error) {
	token, errToken := jwt.Parse(accessToken, func(token *jwt.Token) (any, error) {
		return j.Signature, nil
	})
	if errToken != nil {
		return "", apperrors.ErrInvalidSessionToken
	}
	if types, okType := token.Claims.(jwt.MapClaims)["type"].(string); !okType || types != "session" {
		return "", apperrors.ErrInvalidSessionToken
	}
	if sessionID, okSessionID := token.Claims.(jwt.MapClaims)["sub"].(string); !okSessionID {
		return "", apperrors.ErrInvalidSessionToken
	} else {
		return sessionID, nil
	}
}
