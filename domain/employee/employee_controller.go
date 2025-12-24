package employee

import (
	"net/http"
	"strconv"
	"time"

	"BackendKantorDinsos/infrastructure/database"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type CreateEmployeeRequest struct {
	Name     string `form:"name" binding:"required"`
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type SearchEmployeeRequest struct {
	Name string `form:"name" binding:"required"`
}

func CreateAdminOnce(c *gin.Context) {
	var count int64
	database.DB.Model(&Employee{}).
		Where("role = ?", "admin").
		Count(&count)

	if count > 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin sudah ada",
		})
		return
	}

	var req CreateEmployeeRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid form-data: " + err.Error(),
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to hash password",
		})
		return
	}

	admin := Employee{
		Name:         req.Name,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Role:         "admin",
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat admin",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin pertama berhasil dibuat",
	})
}

// ============================================================================
// CREATE
// ============================================================================
func CreateEmployee(c *gin.Context) {
	var req CreateEmployeeRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid form-data: " + err.Error(),
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to hash password",
		})
		return
	}

	employee := Employee{
		Name:         req.Name,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Role:         "staff", // 🔒 HARD-CODE
	}

	if err := database.DB.Create(&employee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create employee: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Staff berhasil dibuat",
		"employee": employee,
	})
}

// ============================================================================
// SEARCH EMPLOYEE
// ============================================================================
func SearchEmployeeByName(c *gin.Context) {
	name := c.Query("name")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Parameter 'name' tidak boleh kosong",
		})
		return
	}

	var employees []Employee

	if err := database.DB.Where("name LIKE ?", "%"+name+"%").Find(&employees).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mencari employee: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Berhasil mencari employee",
		"employees": employees,
		"count":     len(employees),
	})
}

// ============================================================================
// GET ALL EMPLOYEES
// ============================================================================
func GetAllEmployees(c *gin.Context) {
	// Parse query parameters untuk pagination dan filter
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	name := c.Query("name")
	role := c.Query("role")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := database.DB.Model(&Employee{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if role != "" {
		query = query.Where("role = ?", role)
	}

	var total int64
	query.Count(&total)

	if sortOrder == "desc" {
		query = query.Order(sortBy + " DESC")
	} else {
		query = query.Order(sortBy + " ASC")
	}

	query = query.Offset(offset).Limit(limit)

	var employees []Employee
	if err := query.Find(&employees).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data employee: " + err.Error(),
		})
		return
	}

	type EmployeeResponse struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Username  string    `json:"username"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	var response []EmployeeResponse
	for _, emp := range employees {
		response = append(response, EmployeeResponse{
			ID:        emp.ID,
			Name:      emp.Name,
			Username:  emp.Username,
			Role:      emp.Role,
			CreatedAt: emp.CreatedAt,
			UpdatedAt: emp.UpdatedAt,
		})
	}

	totalPages := total / int64(limit)
	if total%int64(limit) > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data employee",
		"data":    response,
		"meta": gin.H{
			"page":        page,
			"limit":       limit,
			"total_items": total,
			"total_pages": totalPages,
			"has_next":    page < int(totalPages),
			"has_prev":    page > 1,
		},
	})
}

// ============================================================================
// GET MY PROFILE (ME)
// ============================================================================
func GetMe(c *gin.Context) {
	employeeID, exists := c.Get("employeeID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak terautentikasi",
		})
		return
	}

	id, ok := employeeID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Format user ID tidak valid",
		})
		return
	}

	var employee Employee
	if err := database.DB.Where("id = ?", id).First(&employee).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Employee tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data profile: " + err.Error(),
		})
		return
	}

	type ProfileResponse struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Username  string    `json:"username"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	response := ProfileResponse{
		ID:        employee.ID,
		Name:      employee.Name,
		Username:  employee.Username,
		Role:      employee.Role,
		CreatedAt: employee.CreatedAt,
		UpdatedAt: employee.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil profile",
		"profile": response,
	})
}

// ============================================================================
// UPDATE MY PROFILE (ME)
// ============================================================================
type UpdateMeRequest struct {
	Name     string `form:"name"`
	Username string `form:"username"`
}

func UpdateMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak terautentikasi",
		})
		return
	}

	id, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Format user ID tidak valid",
		})
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid form-data: " + err.Error(),
		})
		return
	}

	if req.Name == "" && req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tidak ada data yang akan diupdate",
		})
		return
	}

	var employee Employee
	if err := database.DB.Where("id = ?", id).First(&employee).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Employee tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data employee: " + err.Error(),
		})
		return
	}

	if req.Username != "" && req.Username != employee.Username {
		var count int64
		database.DB.Model(&Employee{}).
			Where("username = ? AND id != ?", req.Username, id).
			Count(&count)

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Username sudah digunakan",
			})
			return
		}
	}

	updateData := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != "" {
		updateData["name"] = req.Name
	}

	if req.Username != "" {
		updateData["username"] = req.Username
	}

	if err := database.DB.Model(&Employee{}).
		Where("id = ?", id).
		Updates(updateData).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengupdate profile: " + err.Error(),
		})
		return
	}

	var updatedEmployee Employee
	database.DB.Where("id = ?", id).First(&updatedEmployee)

	type ProfileResponse struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Username  string    `json:"username"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	response := ProfileResponse{
		ID:        updatedEmployee.ID,
		Name:      updatedEmployee.Name,
		Username:  updatedEmployee.Username,
		Role:      updatedEmployee.Role,
		CreatedAt: updatedEmployee.CreatedAt,
		UpdatedAt: updatedEmployee.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile berhasil diupdate",
		"profile": response,
	})
}

// ============================================================================
// CHANGE MY PASSWORD
// ============================================================================
type ChangePasswordRequest struct {
	CurrentPassword string `form:"current_password" binding:"required"`
	NewPassword     string `form:"new_password" binding:"required"`
	ConfirmPassword string `form:"confirm_password" binding:"required"`
}

func ChangePassword(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak terautentikasi",
		})
		return
	}

	id, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Format user ID tidak valid",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid form-data: " + err.Error(),
		})
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password baru dan konfirmasi password tidak cocok",
		})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password baru minimal 6 karakter",
		})
		return
	}

	var employee Employee
	if err := database.DB.Where("id = ?", id).First(&employee).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Employee tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data employee: " + err.Error(),
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password saat ini salah",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengenkripsi password baru",
		})
		return
	}

	if err := database.DB.Model(&Employee{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"password_hash": string(hashedPassword),
			"updated_at":    time.Now(),
		}).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengubah password: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password berhasil diubah",
	})
}

// ============================================================================
// UPDATE EMPLOYEE (ADMIN ONLY)
// ============================================================================
type UpdateEmployeeRequest struct {
	Name     string `form:"name"`
	Username string `form:"username"`
	Role     string `form:"role"`
}

func UpdateEmployee(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID employee tidak boleh kosong",
		})
		return
	}

	var req UpdateEmployeeRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid form-data: " + err.Error(),
		})
		return
	}

	if req.Name == "" && req.Username == "" && req.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tidak ada data yang akan diupdate. Minimal satu field (name, username, atau role) harus diisi",
		})
		return
	}

	var existingEmployee Employee
	if err := database.DB.Where("id = ?", id).First(&existingEmployee).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Employee tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data employee: " + err.Error(),
		})
		return
	}

	if req.Role != "" {
		validRoles := map[string]bool{
			"admin":      true,
			"staff":      true,
			"supervisor": true,
		}

		if !validRoles[req.Role] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Role tidak valid. Role yang diperbolehkan: admin, staff, supervisor",
			})
			return
		}
	}

	if req.Username != "" && req.Username != existingEmployee.Username {
		var count int64
		database.DB.Model(&Employee{}).
			Where("username = ? AND id != ?", req.Username, id).
			Count(&count)

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Username sudah digunakan oleh employee lain",
			})
			return
		}
	}

	updateData := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != "" {
		updateData["name"] = req.Name
	}

	if req.Username != "" {
		updateData["username"] = req.Username
	}

	if req.Role != "" {
		updateData["role"] = req.Role
	}

	tx := database.DB.Begin()

	if err := tx.Model(&Employee{}).
		Where("id = ?", id).
		Updates(updateData).Error; err != nil {

		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengupdate employee: " + err.Error(),
		})
		return
	}

	var updatedEmployee Employee
	if err := tx.Where("id = ?", id).First(&updatedEmployee).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data employee setelah update: " + err.Error(),
		})
		return
	}

	tx.Commit()

	type EmployeeResponse struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Username  string    `json:"username"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	response := EmployeeResponse{
		ID:        updatedEmployee.ID,
		Name:      updatedEmployee.Name,
		Username:  updatedEmployee.Username,
		Role:      updatedEmployee.Role,
		CreatedAt: updatedEmployee.CreatedAt,
		UpdatedAt: updatedEmployee.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Employee berhasil diupdate",
		"employee": response,
	})
}

// ============================================================================
// DELETE EMPLOYEE (ADMIN ONLY)
// ============================================================================
func DeleteEmployee(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID employee tidak boleh kosong",
		})
		return
	}

	var employee Employee
	if err := database.DB.Where("id = ?", id).First(&employee).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Employee tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data employee: " + err.Error(),
		})
		return
	}

	tx := database.DB.Begin()

	if err := tx.Delete(&Employee{}, "id = ?", id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menghapus employee: " + err.Error(),
		})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Employee berhasil dihapus",
		"data": gin.H{
			"id":       employee.ID,
			"name":     employee.Name,
			"username": employee.Username,
			"role":     employee.Role,
		},
	})
}
