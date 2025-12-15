package main

import (
	"BackendKantorDinsos/domain/employee"
	"BackendKantorDinsos/domain/login"

	"BackendKantorDinsos/infrastructure/database"
	"BackendKantorDinsos/infrastructure/routes"

	"BackendKantorDinsos/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	database.ConnectDatabase()

	database.DB.AutoMigrate(
		&employee.Employee{},
		&login.RefreshToken{},
	)

	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.XSSBlocker())

	routes.EmployeeRoutes(r)
	routes.AuthRoutes(r)
	routes.DocumentStaffRoutes(r)

	r.Run(":8080")
}
