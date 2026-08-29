package integration

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/auth"
	"context"
	"errors"
	"shared/shconstant"
	"shared/storage"
	"shared/testutil"
	"testing"

	"github.com/google/uuid"
)

func TestCreateUserSession_Success(t *testing.T) {
	if errInitial != nil {
		t.Fatal(errInitial)
	}
	testutil.CleanRedis(t, suiteContainer.Redis)
	authRepo := auth.NewRepositoryRedis(&storage.Redis{
		Client: suiteContainer.Redis,
	}, logger)
	sessionUUID := uuid.New().String()
	email := "example@gmail.com"
	if errCreateSession := authRepo.CreateUserSession(context.Background(), sessionUUID, auth.ActionRegister, map[string]any{"email": email}); errCreateSession != nil {
		t.Fatal("failed to create session: ", errCreateSession)
	}
	dataSession, errGetSession := authRepo.GetUserSession(context.Background(), sessionUUID, auth.ActionRegister)
	if errGetSession != nil {
		t.Fatal("failed to get session: ", errGetSession)
	}
	if dataSession["email"] != email {
		t.Fatalf("expected email %s got %s", email, dataSession["email"])
	}
}

func TestGetUserSession_BorderlineCase(t *testing.T) {
	if errInitial != nil {
		t.Fatal(errInitial)
	}
	testutil.CleanRedis(t, suiteContainer.Redis)
	authRepo := auth.NewRepositoryRedis(&storage.Redis{
		Client: suiteContainer.Redis,
	}, logger)
	sessionUUID := uuid.New().String()
	email := "example@gmail.com"
	if errCreateSession := authRepo.CreateUserSession(context.Background(), sessionUUID, auth.ActionRegister, map[string]any{"email": email}); errCreateSession != nil {
		t.Fatal("failed to create session: ", errCreateSession)
	}
	dataSession := make(map[string]string)
	var errGetSession error
	lastTry := 5
	for i := 1; i <= lastTry+1; i++ {
		dataSession, errGetSession = authRepo.GetUserSession(context.Background(), sessionUUID, auth.ActionRegister)
		if i < lastTry {
			if errGetSession != nil {
				t.Fatalf("failed to get session: %v iteration: %d", errGetSession, i)
			}
			if dataSession["email"] != email {
				t.Fatalf("expected email %s got %s iteration: %d", email, dataSession["email"], i)
			}
		}
		if i > lastTry {
			if !errors.Is(errGetSession, apperrors.ErrSessionExpired) {
				t.Fatalf("expected error: %v got %v", apperrors.ErrSessionExpired, errGetSession)
			}
		}
	}
}

func TestCreateRefresh_Success(t *testing.T) {
	if errInitial != nil {
		t.Fatal(errInitial)
	}
	testutil.CleanRedis(t, suiteContainer.Redis)
	authRepo := auth.NewRepositoryRedis(&storage.Redis{
		Client: suiteContainer.Redis,
	}, logger)
	userUUID := uuid.New().String()
	refreshUUID := uuid.New().String()
	ipUser := "185.213.154.12"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	email := "example@gmail.com"
	if errCreateRefresh := authRepo.CreateRefresh(context.Background(), &auth.CreateRefreshParams{
		UserUUID:    userUUID,
		RefreshUUID: refreshUUID,
		EmailUser:   email,
		IPUser:      ipUser,
		UserAgent:   userAgent,
	}); errCreateRefresh != nil {
		t.Fatal("failed to create refresh: ", errCreateRefresh)
	}
	refreshData, _, errGetRefresh := authRepo.GetRefreshData(context.Background(), userUUID, refreshUUID)
	if errGetRefresh != nil {
		t.Fatal("failed to get refresh data: ", errGetRefresh)
	}
	if refreshData.IP != ipUser {
		t.Fatalf("expected ip %s got %s", ipUser, refreshData.IP)
	}
	if refreshData.Email != email {
		t.Fatalf("expected email %s got %s", email, refreshData.Email)
	}
	if refreshData.UserAgent != userAgent {
		t.Fatalf("expected user-agent %s got %s", userAgent, refreshData.UserAgent)
	}
}

func TestLogoutRefresh_Success(t *testing.T) {
	if errInitial != nil {
		t.Fatal(errInitial)
	}
	testutil.CleanRedis(t, suiteContainer.Redis)
	authRepo := auth.NewRepositoryRedis(&storage.Redis{
		Client: suiteContainer.Redis,
	}, logger)
	userUUID := uuid.New().String()
	firstRefreshUUID := uuid.New().String()
	email := "example@gmail.com"
	ip := "185.213.154.12"
	if errCreateRefresh := authRepo.CreateRefresh(context.Background(), &auth.CreateRefreshParams{
		UserUUID:    userUUID,
		RefreshUUID: firstRefreshUUID,
		EmailUser:   email,
		IPUser:      ip,
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	}); errCreateRefresh != nil {
		t.Fatal("failed to create first refresh: ", errCreateRefresh)
	}
	secondRefreshUUID := uuid.New().String()
	secondUserAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15"
	if errCreateRefresh := authRepo.CreateRefresh(context.Background(), &auth.CreateRefreshParams{
		UserUUID:    userUUID,
		RefreshUUID: secondRefreshUUID,
		EmailUser:   email,
		IPUser:      ip,
		UserAgent:   secondUserAgent,
	}); errCreateRefresh != nil {
		t.Fatal("failed to create second refresh: ", errCreateRefresh)
	}
	_, firstRefreshKey, errGetKey := authRepo.GetRefreshData(context.Background(), userUUID, firstRefreshUUID)
	if errGetKey != nil {
		t.Fatal("failed to get key: ", errGetKey)
	}
	if errLogout := authRepo.LogoutRefresh(context.Background(), userUUID, firstRefreshKey); errLogout != nil {
		t.Fatal("failed to logout: ", errLogout)
	}
	if firstDataRefresh, _, errGetFirst := authRepo.GetRefreshData(context.Background(), userUUID, firstRefreshUUID); errGetFirst == nil {
		t.Fatal("failed to logout first refresh: ", firstDataRefresh)
	}
	secondDataRefresh, _, errGetRefresh := authRepo.GetRefreshData(context.Background(), userUUID, secondRefreshUUID)
	if errGetRefresh != nil {
		t.Fatal("failed to get refresh data: ", errGetRefresh)
	}
	if secondDataRefresh.IP != ip {
		t.Fatalf("expected ip %s got %s", ip, secondDataRefresh.IP)
	}
	if secondDataRefresh.Email != email {
		t.Fatalf("expected email %s got %s", email, secondDataRefresh.Email)
	}
	if secondDataRefresh.UserAgent != secondUserAgent {
		t.Fatalf("expected user-agent %s got %s", secondUserAgent, secondDataRefresh.UserAgent)
	}
}

func TestRotationRefresh_Success(t *testing.T) {
	if errInitial != nil {
		t.Fatal(errInitial)
	}
	testutil.CleanRedis(t, suiteContainer.Redis)
	authRepo := auth.NewRepositoryRedis(&storage.Redis{
		Client: suiteContainer.Redis,
	}, logger)
	userUUID := uuid.New().String()
	oldRefreshUUID := uuid.New().String()
	email := "example@gmail.com"
	ip := "185.213.154.12"
	if errCreateRefresh := authRepo.CreateRefresh(context.Background(), &auth.CreateRefreshParams{
		UserUUID:    userUUID,
		RefreshUUID: oldRefreshUUID,
		EmailUser:   email,
		IPUser:      ip,
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	}); errCreateRefresh != nil {
		t.Fatal("failed to create first refresh: ", errCreateRefresh)
	}
	newRefreshUUID := uuid.New().String()
	secondUserAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15"
	if errCreateRefresh := authRepo.CreateRefresh(context.Background(), &auth.CreateRefreshParams{
		UserUUID:    userUUID,
		RefreshUUID: newRefreshUUID,
		EmailUser:   email,
		IPUser:      ip,
		UserAgent:   secondUserAgent,
	}); errCreateRefresh != nil {
		t.Fatal("failed to create second refresh: ", errCreateRefresh)
	}
	newKeyRefresh := newRefreshUUID + auth.NullByte + secondUserAgent + auth.NullByte + ip + auth.NullByte + email
	_, oldRefreshKey, errGetKey := authRepo.GetRefreshData(context.Background(), userUUID, oldRefreshUUID)
	if errGetKey != nil {
		t.Fatal("failed to get key: ", errGetKey)
	}
	if errRotation := authRepo.RotationRefresh(context.Background(), userUUID, newKeyRefresh, oldRefreshKey); errRotation != nil {
		t.Fatal("failed to rotation refresh: ", errRotation)
	}
	if oldDataRefresh, _, errGetFirst := authRepo.GetRefreshData(context.Background(), userUUID, oldRefreshUUID); errGetFirst == nil {
		t.Fatal("failed to rotation: ", oldDataRefresh)
	}
	newDataRefresh, _, errGetRefresh := authRepo.GetRefreshData(context.Background(), userUUID, newRefreshUUID)
	if errGetRefresh != nil {
		t.Fatal("failed to get new refresh data: ", errGetRefresh)
	}
	if newDataRefresh.IP != ip {
		t.Fatalf("expected ip %s got %s", ip, newDataRefresh.IP)
	}
	if newDataRefresh.Email != email {
		t.Fatalf("expected email %s got %s", email, newDataRefresh.Email)
	}
	if newDataRefresh.UserAgent != secondUserAgent {
		t.Fatalf("expected user-agent %s got %s", secondUserAgent, newDataRefresh.UserAgent)
	}
}

func TestDeleteRefresh_Success(t *testing.T) {
	if errInitial != nil {
		t.Fatal(errInitial)
	}
	testutil.CleanRedis(t, suiteContainer.Redis)
	authRepo := auth.NewRepositoryRedis(&storage.Redis{
		Client: suiteContainer.Redis,
	}, logger)
	userUUID := uuid.New().String()
	refreshUUID := uuid.New().String()
	ipUser := "185.213.154.12"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	email := "example@gmail.com"
	if errCreateRefresh := authRepo.CreateRefresh(context.Background(), &auth.CreateRefreshParams{
		UserUUID:    userUUID,
		RefreshUUID: refreshUUID,
		EmailUser:   email,
		IPUser:      ipUser,
		UserAgent:   userAgent,
	}); errCreateRefresh != nil {
		t.Fatal("failed to create refresh: ", errCreateRefresh)
	}
	ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutRedis)
	defer cancel()
	if errDeleteRefresh := authRepo.DeleteUserRefreshes(ctxTimeout, userUUID); errDeleteRefresh != nil {
		t.Fatal("failed to delete refresh: ", errDeleteRefresh)
	}
	keyUserRefreshes := auth.UserRefreshesKey + userUUID
	existRefresh, errCheckExist := suiteContainer.Redis.Exists(context.Background(), keyUserRefreshes).Result()
	if errCheckExist != nil {
		t.Fatal("failed to check user refreshes: ", errCheckExist)
	}
	if existRefresh != 0 {
		t.Fatal("refresh key exists")
	}
}
