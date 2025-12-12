package logout

import (
	"net/http"

	"BackendKantorDinsos/domain/login"
	"BackendKantorDinsos/infrastructure/database"

	"github.com/gin-gonic/gin"
)

type LogoutRequest struct {
	RefreshToken string `form:"refresh_token" binding:"required"`
}

func Logout(c *gin.Context) {
	var req LogoutRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing refresh token"})
		return
	}

	hash := login.HashRefreshToken(req.RefreshToken)

	result := database.DB.Where("token_hash = ?", hash).Unscoped().Delete(&login.RefreshToken{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful (token deleted)",
	})
}
