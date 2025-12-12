package routes

import (
	documentStaffController "BackendKantorDinsos/domain/document_staff"
	"BackendKantorDinsos/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

func DocumentStaffRoutes(r *gin.Engine) {
	ds := r.Group("/api/document_staff", middleware.AuthMiddleware())
	{
		ds.POST("/upload", documentStaffController.CreateDocumentStaff)

		ds.GET("/my-documents", documentStaffController.GetMyDocumentsStaff)

		ds.PATCH("/my-documents/:id", documentStaffController.UpdateMyDocumentStaff)

		ds.DELETE("/:id", documentStaffController.DeleteDocumentStaff)

		adminGroup := ds.Group("")
		adminGroup.Use(middleware.AdminMiddleware())
		{
			adminGroup.POST("/", documentStaffController.CreateDocumentStaffAdmin)

			adminGroup.GET("/", documentStaffController.GetAllDocumentsStaffAdmin)

			adminGroup.PATCH("/:id", documentStaffController.UpdateDocumentStaffAdmin)
		}
	}
}
