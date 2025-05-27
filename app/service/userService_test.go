package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sonartest_cart/app/dto"
	helpermocks "sonartest_cart/app/helper/mocks"
	"sonartest_cart/app/internal"
	internalmocks "sonartest_cart/app/internal/mocks"
	"sonartest_cart/pkg/e"

	jwtmocks "sonartest_cart/pkg/jwt/mocks"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSaveUserDetails(t *testing.T) {
	validBody := []byte(`{"UserName": "testuser", "password": "password123", "mail": "test@example.com", "address": "123 Test St", "pincode": 123456, "phonenumber": 9876543210}`)

	tests := []struct {
		name        string
		rbody       []byte
		parseErr    error
		validateErr error
		saveErr     error
		userID      int64
		want        *dto.SaveUserResponse
		wantErr     bool
		errCode     int
	}{
		{
			name:    "success_case",
			rbody:   validBody,
			userID:  101,
			want:    &dto.SaveUserResponse{UserId: 101},
			wantErr: false,
		},
		{
			name:     "fail_parse_error",
			rbody:    []byte(`invalid-json`),
			parseErr: errors.New("invalid character 'i' looking for beginning of value"),
			wantErr:  true,
			errCode:  e.ErrDecodeRequestBody,
		},
		{
			name:        "fail_validation_error",
			rbody:       []byte(`{"UserName": "testuser"}`), // Missing required fields
			validateErr: errors.New("validation error"),
			wantErr:     true,
			errCode:     e.ErrValidateRequest,
		},
		{
			name:    "fail_save_error",
			rbody:   validBody,
			userID:  0,
			saveErr: errors.New("db error"),
			wantErr: true,
			errCode: e.ErrCreateUser,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create fresh mocks for each test case
			userRepoMock := new(internalmocks.UserRepo)
			helperMock := new(helpermocks.ContextHelper)
			jwtMock := new(jwtmocks.JWTService)
			userService := NewUserService(userRepoMock, helperMock, jwtMock)

			req, _ := http.NewRequest("POST", "/", bytes.NewBuffer(test.rbody))
			req.Header.Set("Content-Type", "application/json")

			// Only mock SaveUserDetails for cases that go that far
			if test.name == "success_case" || test.name == "fail_save_error" {
				fmt.Printf("Mock returning: userID=%d, err=%v\n", test.userID, test.saveErr)
				userRepoMock.On("SaveUserDetails", mock.MatchedBy(func(req *dto.UserDetailSaveRequest) bool {
					return req.UserName == "testuser" && req.Password == "password123"
				})).Return(test.userID, test.saveErr)
			}

			resp, err := userService.SaveUserDetails(req)

			if test.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if wrapErr, ok := err.(*e.WrapError); ok {
					assert.Equal(t, test.errCode, wrapErr.ErrorCode)
				} else {
					t.Errorf("expected error to be of type *e.WrapError, got %T", err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.want, resp)
			}

			userRepoMock.AssertExpectations(t)
		})
	}
}

func TestLoginUser(t *testing.T) {
	tests := []struct {
		name    string
		rbody   []byte
		mock    func(userRepoMock *internalmocks.UserRepo, jwtMock *jwtmocks.JWTService)
		want    *dto.LoginResponse
		wantErr error
	}{
		{
			name:  "fail_decode_request",
			rbody: []byte(`{"invalid": "json"`),
			mock:  func(_ *internalmocks.UserRepo, _ *jwtmocks.JWTService) {},
			want:  nil,
			wantErr: e.NewError(
				e.ErrDecodeRequestBody,
				"error while parsing",
				errors.New("unexpected EOF"),
			),
		},
		{
			name:  "fail_validate_request",
			rbody: []byte(`{"username": "testuser"}`),
			mock:  func(_ *internalmocks.UserRepo, _ *jwtmocks.JWTService) {},
			want:  nil,
			wantErr: e.NewError(
				e.ErrValidateRequest,
				"error while validating",
				errors.New("Key: 'LoginRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag"),
			),
		},
		{
			name:  "fail_user_not_found_db",
			rbody: []byte(`{"username": "testuser", "password": "pass"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, _ *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "testuser").Return(nil, gorm.ErrRecordNotFound).Once()
			},
			want: nil,
			wantErr: e.NewError(
				e.ErrUserNotFound,
				"user not found",
				gorm.ErrRecordNotFound,
			),
		},
		{
			name:  "fail_user_nil",
			rbody: []byte(`{"username": "testuser", "password": "pass"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, _ *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "testuser").Return(nil, nil).Once()
			},
			want: nil,
			wantErr: e.NewError(
				e.ErrUserNotFound,
				"user not found",
				nil,
			),
		},
		{
			name:  "fail_db_error",
			rbody: []byte(`{"username": "testuser", "password": "pass"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, _ *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "testuser").Return(nil, errors.New("some db error")).Once()
			},
			want: nil,
			wantErr: e.NewError(
				e.ErrLoginUser,
				"error during login",
				errors.New("some db error"),
			),
		},
		{
			name:  "fail_wrong_password",
			rbody: []byte(`{"username": "testuser", "password": "wrong"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, _ *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "testuser").Return(&internal.Userdetail{
					ID:       1,
					Username: "testuser",
					Password: "correct",
					Status:   true,
				}, nil).Once()
			},
			want: nil,
			wantErr: e.NewError(
				e.ErrInvalidCredentials,
				"invalid password",
				fmt.Errorf("invalid password for user %s", "testuser"),
			),
		},
		{
			name:  "fail_user_blocked",
			rbody: []byte(`{"username": "testuser", "password": "password"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, _ *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "testuser").Return(&internal.Userdetail{
					ID:       1,
					Username: "testuser",
					Password: "password",
					Status:   false,
				}, nil).Once()
			},
			want: nil,
			wantErr: e.NewError(
				e.ErrUserBlocked,
				"user is blocked",
				fmt.Errorf("user %s is blocked", "testuser"),
			),
		},
		{
			name:  "fail_token_generation",
			rbody: []byte(`{"username": "testuser", "password": "password"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, jwtMock *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "testuser").Return(&internal.Userdetail{
					ID:       1,
					Username: "testuser",
					Password: "password",
					Status:   true,
					IsAdmin:  false,
				}, nil).Once()
				jwtMock.On("GenerateToken", int64(1), "testuser", false).
					Return("", errors.New("token error")).Once()
			},
			want: nil,
			wantErr: e.NewError(
				e.ErrGenerateToken,
				"failed to generate token",
				errors.New("token error"),
			),
		},
		{
			name:  "success_login",
			rbody: []byte(`{"username": "testuser", "password": "password"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, jwtMock *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "testuser").Return(&internal.Userdetail{
					ID:       1,
					Username: "testuser",
					Password: "password",
					Status:   true,
					IsAdmin:  false,
				}, nil).Once()
				jwtMock.On("GenerateToken", int64(1), "testuser", false).
					Return("mocked-token", nil).Once()
			},
			want: &dto.LoginResponse{
				Token: "mocked-token",
			},
			wantErr: nil,
		},
		{
			name:  "success_login_admin",
			rbody: []byte(`{"username": "adminuser", "password": "adminpass"}`),
			mock: func(userRepoMock *internalmocks.UserRepo, jwtMock *jwtmocks.JWTService) {
				userRepoMock.On("GetUserByUsername", "adminuser").Return(&internal.Userdetail{
					ID:       2,
					Username: "adminuser",
					Password: "adminpass",
					Status:   true,
					IsAdmin:  true,
				}, nil).Once()
				jwtMock.On("GenerateToken", int64(2), "adminuser", true).
					Return("admin-token", nil).Once()
			},
			want: &dto.LoginResponse{
				Token: "admin-token",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepoMock := internalmocks.NewUserRepo(t)
			contextHelperMock := helpermocks.NewContextHelper(t)
			jwtMock := jwtmocks.NewJWTService(t)

			userService := NewUserService(userRepoMock, contextHelperMock, jwtMock)
			tt.mock(userRepoMock, jwtMock)

			req := httptest.NewRequest("POST", "/", bytes.NewReader(tt.rbody))
			req.Header.Set("Content-Type", "application/json")

			got, err := userService.LoginUser(req)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, got)
				assert.Equal(t, tt.wantErr.(*e.WrapError).ErrorCode, err.(*e.WrapError).ErrorCode)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.want.Token, got.Token)
			}

			userRepoMock.AssertExpectations(t)
			jwtMock.AssertExpectations(t)
		})
	}
}

func TestGetUserIDAndCheckStatus(t *testing.T) {
	type mocks struct {
		helper   *helpermocks.ContextHelper
		userRepo *internalmocks.UserRepo
		jwt      *jwtmocks.JWTService
	}

	tests := []struct {
		name         string
		ctx          context.Context
		mockSetup    func(m mocks)
		wantUserID   int64
		wantErr      bool
		expectedCode int
	}{
		{
			name: "success_case",
			ctx:  context.Background(),
			mockSetup: func(m mocks) {
				m.helper.On("GetUserID", mock.Anything).Return(int64(123), nil)
				m.userRepo.On("IsUserActive", int64(123)).Return(true, nil)
			},
			wantUserID: 123,
			wantErr:    false,
		},
		{
			name: "fail_get_user_id",
			ctx:  context.Background(),
			mockSetup: func(m mocks) {
				m.helper.On("GetUserID", mock.Anything).Return(int64(0), errors.New("context error"))
			},
			wantUserID:   0,
			wantErr:      true,
			expectedCode: e.ErrContextError,
		},
		{
			name: "fail_is_user_active_error",
			ctx:  context.Background(),
			mockSetup: func(m mocks) {
				m.helper.On("GetUserID", mock.Anything).Return(int64(456), nil)
				m.userRepo.On("IsUserActive", int64(456)).Return(false, errors.New("db failure"))
			},
			wantUserID:   0,
			wantErr:      true,
			expectedCode: e.ErrGetUserDetails,
		},
		{
			name: "fail_user_not_active",
			ctx:  context.Background(),
			mockSetup: func(m mocks) {
				m.helper.On("GetUserID", mock.Anything).Return(int64(789), nil)
				m.userRepo.On("IsUserActive", int64(789)).Return(false, nil)
			},
			wantUserID:   0,
			wantErr:      true,
			expectedCode: e.ErrUserBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh mocks per test
			helperMock := new(helpermocks.ContextHelper)
			userRepoMock := new(internalmocks.UserRepo)
			jwtMock := new(jwtmocks.JWTService)

			userService := &userServiceImpl{
				userRepo:      userRepoMock,
				contextHelper: helperMock,
				jwtService:    jwtMock,
			}

			tt.mockSetup(mocks{
				helper:   helperMock,
				userRepo: userRepoMock,
				jwt:      jwtMock,
			})

			gotUserID, err := userService.getUserIDAndCheckStatus(tt.ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, int64(0), gotUserID)
				if wrapErr, ok := err.(*e.WrapError); ok {
					assert.Equal(t, tt.expectedCode, wrapErr.ErrorCode)
				} else {
					t.Errorf("expected WrapError, got %T", err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUserID, gotUserID)
			}

			helperMock.AssertExpectations(t)
			userRepoMock.AssertExpectations(t)
		})
	}
}
