package controller

import (
	"sonartest_cart/app/service"
	"sonartest_cart/pkg/api"
	"sonartest_cart/pkg/e"
	"net/http"
)

type UserController interface {
	LoginUser(w http.ResponseWriter, r *http.Request)
	UserDetails(w http.ResponseWriter, r *http.Request)
}

type UserControllerImpl struct {
	userService service.UserService
}

func NewUserController(userService service.UserService) UserController {
	return &UserControllerImpl{
		userService: userService,
	}
}

func (c *UserControllerImpl) UserDetails(w http.ResponseWriter, r *http.Request) {
	resp, err := c.userService.SaveUserDetails(r)
	if err != nil {
		apiErr := e.NewAPIError(err, "failed to create user")
		api.Fail(w, apiErr.StatusCode, apiErr.Code, apiErr.Message, err.Error())
		return
	}
	api.Success(w, http.StatusOK, resp)
}

func (c *UserControllerImpl) LoginUser(w http.ResponseWriter, r *http.Request) {
	resp, err := c.userService.LoginUser(r)
	if err != nil {
		apiErr := e.NewAPIError(err, "failed to login user")
		api.Fail(w, apiErr.StatusCode, apiErr.Code, apiErr.Message, err.Error())
		return
	}
	api.Success(w, http.StatusOK, resp)
}
