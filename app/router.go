package app

import (
	"sonartest_cart/app/controller"
	"sonartest_cart/app/helper"
	"sonartest_cart/app/internal"
	"sonartest_cart/app/service"
	api "sonartest_cart/pkg/api"
	"sonartest_cart/pkg/jwt"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func APIRouter(db *gorm.DB) chi.Router {
	r := chi.NewRouter()

	// User part
	urRepo := internal.NewUserRepo(db)
	hlRepo := helper.NewContextHelper()
	jwtService := jwt.NewJWTService
	urService := service.NewUserService(urRepo, hlRepo, jwtService())
	urController := controller.NewUserController(urService)

	r.Route("/", func(r chi.Router) {
		r.Get("/hello", api.ExampleHamdler)
		r.Post("/signup", urController.UserDetails)
		r.Post("/login", urController.LoginUser)
	})

	return r
}
