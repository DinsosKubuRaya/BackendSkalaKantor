package login

import (
	"net/http"
	"time"

	"BackendKantorDinsos/domain/employee"
	"BackendKantorDinsos/infrastructure/database"

	"github.com/gin-gonic/gin"
)

type RefreshRequest struct {
	RefreshToken string `form:"refresh_token" binding:"required"`
}

func Refresh(c *gin.Context) {
	var req RefreshRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing refresh token"})
		return
	}

	hash := HashRefreshToken(req.RefreshToken)

	var rt RefreshToken
	err := database.DB.Where("token_hash = ? AND revoked = false", hash).First(&rt).Error
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	if time.Now().After(rt.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired"})
		return
	}

	var user employee.Employee
	if err := database.DB.Where("id = ?", rt.EmployeeID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	accessToken, _ := GenerateAccessToken(user.ID, user.Role)

	newPlainRefresh := GenerateRefreshTokenString()
	newHash := HashRefreshToken(newPlainRefresh)

	rt.TokenHash = newHash
	rt.ExpiresAt = time.Now().Add(RefreshTokenTTL)
	database.DB.Save(&rt)

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newPlainRefresh,
		ExpiresIn:    int64(AccessTokenTTL.Seconds()),
	})
}
