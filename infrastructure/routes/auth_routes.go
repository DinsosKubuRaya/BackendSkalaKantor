package routes

import (
	"BackendKantorDinsos/domain/login"
	"BackendKantorDinsos/domain/logout"
	"BackendKantorDinsos/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(r *gin.Engine) {
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", login.Login)

		auth.POST("/refresh", login.Refresh)

		auth.POST("/logout", middleware.AuthMiddleware(), logout.Logout)
	}
}
