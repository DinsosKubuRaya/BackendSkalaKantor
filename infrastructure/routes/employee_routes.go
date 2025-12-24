package routes

import (
	employeeController "BackendKantorDinsos/domain/employee"
	"BackendKantorDinsos/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

func EmployeeRoutes(r *gin.Engine) {

	r.POST("/api/setup/admin", employeeController.CreateAdminOnce)

	emp := r.Group("/api/employee", middleware.AuthMiddleware())
	{
		adminGroup := emp.Group("")
		adminGroup.Use(middleware.AdminMiddleware())
		{
			adminGroup.POST("/", employeeController.CreateEmployee)

			adminGroup.GET("/search", employeeController.SearchEmployeeByName)

			adminGroup.GET("/", employeeController.GetAllEmployees)

			adminGroup.PATCH("/:id", employeeController.UpdateEmployee)

			adminGroup.DELETE("/:id", employeeController.DeleteEmployee)
		}

		emp.GET("/me", employeeController.GetMe)

		emp.PATCH("/me", employeeController.UpdateMe)

		emp.PATCH("/me/change-password", employeeController.ChangePassword)
	}
}
