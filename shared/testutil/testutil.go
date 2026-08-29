package testutil

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func RefreshUserTestData(sqlFileData []byte, deleteTableNames []string, t *testing.T) *gorm.DB {
	if errFileEnvTest := godotenv.Load(".env.test"); errFileEnvTest != nil {
		t.Fatal(".env.test file not found")
	}
	dsnTest := os.Getenv("DSN")
	if dsnTest == "" {
		t.Fatal("environment variable 'DSN' not found")
	}
	db, errOpen := gorm.Open(postgres.Open(dsnTest))
	if errOpen != nil {
		t.Fatal("failed to connect PostgreSQL: ", errOpen)
	}
	if errDelete := db.Exec("TRUNCATE " + strings.Join(deleteTableNames, ",")).Error; errDelete != nil {
		t.Fatal("failed to delete old data: ", errDelete)
	}
	if errLoad := db.Exec(string(sqlFileData)).Error; errLoad != nil {
		t.Fatal("failed to load new data: ", errLoad)
	}
	return db
}

type TestResponse[T any] struct {
	Success bool
	Data    T                 `json:"data,omitempty"`
	Error   map[string]string `json:"errors,omitempty"`
}

func HelperHandleResponse[T any](resp *http.Response, statusCode int, t *testing.T) T {
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			t.Fatal("failed to close response: ", errClose)
		}
	}()
	var payload TestResponse[T]
	if statusCode != http.StatusNoContent {
		if errDecode := json.NewDecoder(resp.Body).Decode(&payload); errDecode != nil {
			t.Fatal("failed to decode response: " + errDecode.Error())
		}
		if !payload.Success {
			t.Error("response isn't successful")
		}
	}
	if resp.StatusCode != statusCode {
		t.Fatalf("expected %d got %d", statusCode, resp.StatusCode)
	}
	return payload.Data
}
func CreateTestAccessToken(userUUID, signature string, t *testing.T) string {
	claim := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"type": "access",
		"sub":  userUUID,
		"exp":  time.Now().Add(time.Minute * 5).Unix(),
	})
	token, errToken := claim.SignedString([]byte(signature))
	if errToken != nil {
		t.Fatal("failed to sign token: ", errToken)
		return ""
	}
	return token
}
