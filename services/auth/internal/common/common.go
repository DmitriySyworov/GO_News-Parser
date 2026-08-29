package common

import (
	"time"
)

const (
	TTLRefreshKey = time.Hour * 720
	TTLSessionJWT = time.Minute * 5
	TTLAccessKey  = time.Minute * 15
	CodeKey       = "code"

	AttemptsLeft = "attempts_left"
)

type RefreshData struct {
	UserAgent string
	IP        string
	Email     string
}
type ResponseAuth struct {
	Message    string
	SessionJwt string `json:"session_jwt"`
}
