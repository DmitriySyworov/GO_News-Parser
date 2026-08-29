package main

import (
	"app/auth-service/internal/common"
	"app/auth-service/internal/model"
	"app/auth-service/internal/user"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"shared/shconstant"
	"shared/testutil"
	"testing"
)

func TestGetUserSuccessful(t *testing.T) {
	const userGetUUID = "7b3e1f4a-6d2c-4b8a-9e1c-5f6a7b8c9d0e"
	dataSqlFile, errReadFile := os.ReadFile("load_mock_users.sql")
	if errReadFile != nil {
		t.Fatal("failed to read sql file: ", errReadFile)
	}
	testutil.RefreshUserTestData(dataSqlFile, []string{"users"}, t)
	appVariable := App()
	accessToken := testutil.CreateTestAccessToken(userGetUUID, appVariable.Conf.Signature, t)
	testServer := httptest.NewServer(appVariable.HandlerApp)
	defer testServer.Close()
	request, errReq := http.NewRequest(http.MethodGet, testServer.URL+"/api/v1/user", nil)
	if errReq != nil {
		t.Fatal("failed to prepare request: ", errReq)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	respGet, errRespGet := http.DefaultClient.Do(request)
	if errRespGet != nil {
		t.Fatal("failed to get response: ", errRespGet)
	}
	userResp := testutil.HelperHandleResponse[user.ResponseUser](respGet, http.StatusOK, t)
	if userResp.UserUUID != userGetUUID {
		t.Fatalf("expected uuid %s got %s", userGetUUID, userResp.UserUUID)
	}
}

const (
	NewName     = "newName"
	NewEmail    = "newemail@gmail.com"
	NewPassword = "n(&*^)word234Qw2"
	Email       = "exampleupdate@gmail.com"
)

var CaseDataUpdate = []user.RequestUpdateUser{
	{NewName: NewName},
	{NewName: NewName, NewPassword: NewPassword, Password: testPassword, Email: Email},
	{NewPassword: NewPassword, Password: testPassword, Email: Email},
	{NewEmail: NewEmail, NewPassword: NewPassword, Password: testPassword},
	{NewName: NewName, NewEmail: NewEmail, NewPassword: NewPassword, Password: testPassword},
}

func TestUpdateUserSuccessful(t *testing.T) {
	const userUpdateUUID = "f7b3a4c1-8d2e-4b9a-9e1c-5f6a7b8c9d0e"
	appVariable := App()
	testServer := httptest.NewServer(appVariable.HandlerApp)
	defer testServer.Close()
	dataSqlFile, errReadFile := os.ReadFile("load_mock_users.sql")
	if errReadFile != nil {
		t.Fatal("failed to read sql file: ", errReadFile)
	}
	for _, test := range CaseDataUpdate {
		testutil.RefreshUserTestData(dataSqlFile, []string{"users"}, t)
		deleteRedisData(t)
		deleteMailPitMessages(t)
		accessToken := testutil.CreateTestAccessToken(userUpdateUUID, appVariable.Conf.Signature, t)
		data, errMarshalUpdate := json.Marshal(test)
		if errMarshalUpdate != nil {
			t.Fatal("failed to prepare request: ", errMarshalUpdate)
		}
		request, errReq := http.NewRequest(http.MethodPatch, testServer.URL+"/api/v1/user", bytes.NewBuffer(data))
		if errReq != nil {
			t.Fatal("failed to prepare request: ", errReq)
		}
		request.Header.Set("Authorization", "Bearer "+accessToken)
		respUpdate, errRespUpdate := http.DefaultClient.Do(request)
		if errRespUpdate != nil {
			t.Fatal("failed to get response: ", errRespUpdate)
		}
		if test.NewName != "" && test.NewEmail == "" && test.NewPassword == "" {
			dataResp := testutil.HelperHandleResponse[model.Users](respUpdate, http.StatusOK, t)
			if dataResp.Name != NewName {
				t.Fatalf("expected name %s got %s", NewName, dataResp.Name)
			}
		} else {
			dataResp := testutil.HelperHandleResponse[common.ResponseAuth](respUpdate, http.StatusAccepted, t)
			if dataResp.SessionJwt == "" {
				t.Fatal("sessionJwt is empty")
			}
			code := helperExtractCode(t)
			bodyConfirm := user.RequestConfirm{
				Code: code,
			}
			dataConfirm, errMarshalConfirm := json.Marshal(bodyConfirm)
			if errMarshalConfirm != nil {
				t.Fatal("failed to prepare request Confirm: ", errMarshalConfirm)
			}
			requestConfirm, errReqConfirm := http.NewRequest(http.MethodPost, testServer.URL+"/api/v1/user/confirm?action=update", bytes.NewBuffer(dataConfirm))
			if errReqConfirm != nil {
				t.Fatal("failed to prepare request: ", errReqConfirm)
			}
			requestConfirm.Header.Set("X-Session-Token", "Bearer "+dataResp.SessionJwt)
			requestConfirm.Header.Set("Authorization", "Bearer "+accessToken)
			respConfirm, errRespConfirm := http.DefaultClient.Do(requestConfirm)
			if errRespConfirm != nil {
				t.Fatal("failed to get response confirm: ", errRespConfirm)
			}
			userUpdate := testutil.HelperHandleResponse[user.ResponseUser](respConfirm, http.StatusOK, t)
			if userUpdate.UserUUID != userUpdateUUID {
				t.Fatalf("expected user_uuid %s got %s", userUpdateUUID, userUpdate.UserUUID)
			}
		}
	}
}

var CaseDataRemove = []struct {
	Name     string
	Type     string
	UserUUID string
	user.RequestRemoveUser
}{
	{Name: "hard-delete user - ", RequestRemoveUser: user.RequestRemoveUser{Email: "exampledelete@gmail.com", Password: testPassword}, Type: shconstant.TypeHardDelete, UserUUID: "5a1f4b3e-2c7d-491c-a3f5-6b2d8e1c9a4f"},
	{Name: "soft-delete user - ", RequestRemoveUser: user.RequestRemoveUser{Email: "exampleremove@gmail.com", Password: testPassword}, Type: shconstant.TypeSoftDelete, UserUUID: "9f8e7d6c-5b4a-4321-a1b2-c3d4e5f6a7b8"},
}

func TestRemoveUserSuccessful(t *testing.T) {
	appVariable := App()
	testServer := httptest.NewServer(appVariable.HandlerApp)
	defer testServer.Close()
	dataSqlFile, errReadFile := os.ReadFile("load_mock_users.sql")
	if errReadFile != nil {
		t.Fatal("failed to read sql file: ", errReadFile)
	}
	for _, test := range CaseDataRemove {
		db := testutil.RefreshUserTestData(dataSqlFile, []string{"users"}, t)
		deleteRedisData(t)
		deleteMailPitMessages(t)
		accessToken := testutil.CreateTestAccessToken(test.UserUUID, appVariable.Conf.Signature, t)
		data, errMarshalRemove := json.Marshal(test.RequestRemoveUser)
		if errMarshalRemove != nil {
			t.Fatal(test.Name+"failed to prepare request: ", errMarshalRemove)
		}
		request, errReq := http.NewRequest(http.MethodDelete, testServer.URL+"/api/v1/user?type="+test.Type, bytes.NewBuffer(data))
		if errReq != nil {
			t.Fatal(test.Name+"failed to prepare request: ", errReq)
		}
		request.Header.Set("Authorization", "Bearer "+accessToken)
		respDelete, errRespDelete := http.DefaultClient.Do(request)
		if errRespDelete != nil {
			t.Fatal(test.Name+"failed to get response: ", errRespDelete)
		}
		dataResp := testutil.HelperHandleResponse[common.ResponseAuth](respDelete, http.StatusAccepted, t)
		if dataResp.SessionJwt == "" {
			t.Fatal(test.Name + "sessionJwt is empty")
		}
		code := helperExtractCode(t)
		bodyConfirm := user.RequestConfirm{
			Code: code,
		}
		dataConfirm, errMarshalConfirm := json.Marshal(bodyConfirm)
		if errMarshalConfirm != nil {
			t.Fatal(test.Name+"failed to prepare request Confirm: ", errMarshalConfirm)
		}
		requestConfirm, errReqConfirm := http.NewRequest(http.MethodPost, testServer.URL+"/api/v1/user/confirm?action="+test.Type, bytes.NewBuffer(dataConfirm))
		if errReqConfirm != nil {
			t.Fatal(test.Name+"failed to prepare request: ", errReqConfirm)
		}
		requestConfirm.Header.Set("X-Session-Token", "Bearer "+dataResp.SessionJwt)
		requestConfirm.Header.Set("Authorization", "Bearer "+accessToken)
		respConfirm, errRespConfirm := http.DefaultClient.Do(requestConfirm)
		if errRespConfirm != nil {
			t.Fatal(test.Name+"failed to get response confirm: ", errRespConfirm)
		}
		testutil.HelperHandleResponse[struct{}](respConfirm, http.StatusNoContent, t)
		if test.Type == shconstant.TypeSoftDelete {
			if db.Where("user_uuid = ?", test.UserUUID).
				Take(&model.Users{}).Error == nil {
				t.Fatal(test.Name + "failed to remove user")
			}
		} else if test.Type == shconstant.TypeHardDelete {
			if db.Unscoped().Where("user_uuid = ?", test.UserUUID).
				Take(&model.Users{}).Error == nil {
				t.Fatal(test.Name + "failed to delete user")
			}
		}
	}
}
