package controller

import (
	"errors"
	"sonartest_cart/app/dto"
	"sonartest_cart/app/service/mocks"
	"sonartest_cart/pkg/e"

	"net/http/httptest"
	"testing"

	"github.com/go-playground/assert/v2"
)

func TestUserDetails(t *testing.T) {
	userMock := new(mocks.UserService)
	con := NewUserController(userMock)

	tests := []struct {
		name    string
		status  int
		want    string
		profile *dto.SaveUserResponse
		Error   error
		wantErr bool
	}{
		{
			name:   "success_case",
			status: 200,
			profile: &dto.SaveUserResponse{
				UserId: 1,
			},
			want:    `{"status":"ok","result":{"userid":1}}`,
			wantErr: false,
		},
		{
			name:   "fail_user_details",
			Error:  e.NewError(400, "Bad Request", errors.New("Invalid Request")),
			status: 400,
			//want:   `{"status":"notok","error":{"code":400,"message":"Bad Request","details":["Invalid Request"]}}`,
			want: `{"status":"notok","error":{"code":400,"message":"failed to create user","details":["Invalid Request"]}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)

			// Mocking `SaveUserDetails()` of Service mock
			userMock.Mock.On("SaveUserDetails", req).Once().Return(test.profile, test.Error)

			// calling the function
			con.UserDetails(res, req)

			// comparing the response code and body with expected
			assert.Equal(t, test.status, res.Code)
			assert.Equal(t, test.want, res.Body.String())
		})
	}
}

func TestLoginUser(t *testing.T) {
	userMock := new(mocks.UserService)
	con := NewUserController(userMock)

	tests := []struct {
		name    string
		status  int
		details *dto.LoginResponse
		want    string
		Error   error
		wantErr bool
	}{
		{
			name:   "success_case",
			status: 200,
			details: &dto.LoginResponse{
				Token: "ugewluiglhjFBEWGLIUEHFjkhfEIFJjksdbgrwhgoiwehg",
			},
			want:    `{"status":"ok","result":{"token":"ugewluiglhjFBEWGLIUEHFjkhfEIFJjksdbgrwhgoiwehg"}}`,
			wantErr: false,
		},
		{
			name:   "fail_login",
			Error:  e.NewError(400, "Bad Request", errors.New("Invalid Request")),
			status: 400,
			//want:   `{"status":"nok","error":{"code":400,"message":"Bad Request","details":["Invalid Request"]}}`,
			want: `{"status":"notok","error":{"code":400,"message":"failed to login user","details":["Invalid Request"]}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)

			// Mocking `LoginUser()` of Service mock
			userMock.Mock.On("LoginUser", req).Once().Return(test.details, test.Error)

			// calling the function
			con.LoginUser(res, req)

			// comparing the response code and body with expected
			assert.Equal(t, test.status, res.Code)
			assert.Equal(t, test.want, res.Body.String())
		})
	}
}
