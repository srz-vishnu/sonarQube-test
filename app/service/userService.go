package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sonartest_cart/app/dto"
	helper "sonartest_cart/app/helper"
	"sonartest_cart/app/internal"
	"sonartest_cart/pkg/e"
	"sonartest_cart/pkg/jwt"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type UserService interface {
	SaveUserDetails(r *http.Request) (*dto.SaveUserResponse, error)
	LoginUser(r *http.Request) (*dto.LoginResponse, error)
}

type userServiceImpl struct {
	userRepo      internal.UserRepo
	contextHelper helper.ContextHelper
	jwtService    jwt.JWTService
}

func NewUserService(userRepo internal.UserRepo, ctxHelper helper.ContextHelper, jwtService jwt.JWTService) UserService {
	return &userServiceImpl{
		userRepo:      userRepo,
		contextHelper: ctxHelper,
		jwtService:    jwtService,
	}
}

func (s *userServiceImpl) getUserIDAndCheckStatus(ctx context.Context) (int64, error) {
	userID, err := s.contextHelper.GetUserID(ctx)
	if err != nil {
		return 0, e.NewError(e.ErrContextError, "error while getting userId from ctx", err)
	}
	log.Info().Msgf("userId of the user logged in %d", userID)

	isActive, err := s.userRepo.IsUserActive(userID)
	if err != nil {
		return 0, e.NewError(e.ErrGetUserDetails, "error while checking user details", err)
	}

	if !isActive {
		log.Info().Msg("User is not active.")
		return 0, e.NewError(e.ErrUserBlocked, "user is blocked or inactive", nil)
	}
	log.Info().Msg("User is active")

	return userID, nil
}

func (s *userServiceImpl) SaveUserDetails(r *http.Request) (*dto.SaveUserResponse, error) {
	args := &dto.UserDetailSaveRequest{}

	// parsing the req.body
	err := args.Parse(r)
	if err != nil {
		return nil, e.NewError(e.ErrDecodeRequestBody, "error while parsing", err)
	}

	//validation
	err = args.Validate()
	if err != nil {
		return nil, e.NewError(e.ErrValidateRequest, "error while validating", err)
	}
	log.Info().Msg("Successfully completed parsing and validation of request body")

	userID, err := s.userRepo.SaveUserDetails(args)
	if err != nil {
		return nil, e.NewError(e.ErrCreateUser, "error while creating user", err)
	}
	log.Info().Msgf("Successfully created user with id %d", userID)

	return &dto.SaveUserResponse{
		UserId: userID,
	}, nil
}

func (s *userServiceImpl) LoginUser(r *http.Request) (*dto.LoginResponse, error) {
	args := &dto.LoginRequest{}

	// parsing the req.body
	err := args.Parse(r)
	if err != nil {
		return nil, e.NewError(e.ErrDecodeRequestBody, "error while parsing", err)
	}

	//validation
	err = args.Validate()
	if err != nil {
		return nil, e.NewError(e.ErrValidateRequest, "error while validating", err)
	}
	log.Info().Msg("Successfully completed parsing and validation of request body")

	// Fetching user from database
	user, err := s.userRepo.GetUserByUsername(args.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.NewError(e.ErrUserNotFound, "user not found", err)
		}
		return nil, e.NewError(e.ErrLoginUser, "error during login", err)
	}

	// Check if user is nil
	if user == nil {
		return nil, e.NewError(e.ErrUserNotFound, "user not found", err)
	}

	if user.IsAdmin {
		log.Info().Msg("the user is an admin")
	} else {
		log.Info().Msg("the user is a regular user")
	}

	// Validate password
	if user.Password != args.Password {
		err := fmt.Errorf("invalid password for user %s", user.Username)
		return nil, e.NewError(e.ErrInvalidCredentials, "invalid password", err)
	}

	// Check if user is active
	if !user.Status {
		err := fmt.Errorf("user %s is blocked", user.Username)
		return nil, e.NewError(e.ErrUserBlocked, "user is blocked", err)
	}

	// Generating JWT Token with isAdmin from database
	// token, err := jwt.NewJWTService().GenerateToken(user.ID, user.Username, user.IsAdmin)
	token, err := s.jwtService.GenerateToken(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		return nil, e.NewError(e.ErrGenerateToken, "failed to generate token", err)
	}
	log.Info().Msgf("Generated token for user %s (Admin: %v)", user.Username, user.IsAdmin)

	return &dto.LoginResponse{
		Token: token,
	}, nil
}
