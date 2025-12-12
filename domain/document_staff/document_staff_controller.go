package document_staff

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"BackendKantorDinsos/infrastructure/config"
	"BackendKantorDinsos/infrastructure/database"

	"BackendKantorDinsos/domain/employee"
	"BackendKantorDinsos/domain/user"

	"github.com/gin-gonic/gin"
)

// ======================================================
// CREATE DOCUMENT STAFF - ADMIN ONLY
// ======================================================
func CreateDocumentStaffAdmin(c *gin.Context) {

	userID := c.PostForm("user_id")
	employeeID := c.PostForm("employee_id")
	subject := c.PostForm("subject")

	if subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject wajib diisi"})
		return
	}

	if userID == "" && employeeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UserID atau EmployeeID harus diisi"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
		return
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
		return
	}
	reader := bytes.NewReader(fileBytes)

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	var resourceType, folder string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		resourceType = "image"
		folder = "gambar"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		resourceType = "raw"
		folder = "document_staff"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
		return
	}

	if userID != "" {
		var user user.User
		if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "UserID tidak ditemukan"})
			return
		}
	}

	if employeeID != "" {
		var employee employee.Employee
		if err := database.DB.First(&employee, "id = ?", employeeID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "EmployeeID tidak ditemukan"})
			return
		}
	}

	uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
		return
	}

	document := DocumentStaff{
		UserID:       userID,
		EmployeeID:   employeeID,
		Subject:      subject,
		FileName:     fileHeader.Filename,
		FileURL:      uploadResult.SecureURL,
		PublicID:     uploadResult.PublicID,
		ResourceType: resourceType,
	}

	if err := database.DB.Create(&document).Error; err != nil {
		config.DeleteFromCloudinary(uploadResult.PublicID, resourceType)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Dokumen berhasil dibuat",
		"document": document,
	})
}

// ======================================================
// CREATE DOCUMENT STAFF - FOR LOGGED IN USER
// ======================================================
func CreateDocumentStaff(c *gin.Context) {
	employeeIDRaw, exists := c.Get("employeeID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - employeeID not found"})
		return
	}

	employeeID, ok := employeeIDRaw.(string)
	if !ok || employeeID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid employeeID"})
		return
	}

	var emp employee.Employee
	if err := database.DB.First(&emp, "id = ?", employeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee tidak ditemukan"})
		return
	}

	subject := c.PostForm("subject")
	if subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject wajib diisi"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
		return
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
		return
	}
	reader := bytes.NewReader(fileBytes)

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	var resourceType, folder string

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		resourceType = "image"
		folder = "gambar"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		resourceType = "raw"
		folder = "document_staff"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
		return
	}

	uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
		return
	}

	document := DocumentStaff{
		EmployeeID:   employeeID,
		Subject:      subject,
		FileName:     fileHeader.Filename,
		FileURL:      uploadResult.SecureURL,
		PublicID:     uploadResult.PublicID,
		ResourceType: resourceType,
	}

	if err := database.DB.Create(&document).Error; err != nil {
		config.DeleteFromCloudinary(uploadResult.PublicID, resourceType)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan dokumen: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Dokumen berhasil diupload",
		"document": document,
	})
}

// ======================================================
// GET ALL DOCUMENTS STAFF - ADMIN ONLY
// ======================================================
func GetAllDocumentsStaffAdmin(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "20")
	subject := c.Query("subject")
	userID := c.Query("user_id")
	employeeID := c.Query("employee_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 {
		limitInt = 20
	}

	if limitInt > 100 {
		limitInt = 100
	}

	offset := (pageInt - 1) * limitInt

	query := database.DB.Model(&DocumentStaff{}).
		Select(`document_staffs.id,
				document_staffs.user_id,
				document_staffs.employee_id,
				document_staffs.file_url,
				document_staffs.subject,
				document_staffs.file_name,
				document_staffs.public_id,
				document_staffs.resource_type,
				document_staffs.created_at,
				document_staffs.updated_at,
				CASE 
					WHEN document_staffs.user_id IS NOT NULL AND document_staffs.user_id != '' THEN users.name
					WHEN document_staffs.employee_id IS NOT NULL AND document_staffs.employee_id != '' THEN employees.name
					ELSE ''
				END as owner_name`).
		Joins("LEFT JOIN users ON users.id = document_staffs.user_id").
		Joins("LEFT JOIN employees ON employees.id = document_staffs.employee_id")

	if subject != "" {
		query = query.Where("document_staffs.subject LIKE ?", "%"+subject+"%")
	}

	if userID != "" {
		query = query.Where("document_staffs.user_id = ?", userID)
	}

	if employeeID != "" {
		query = query.Where("document_staffs.employee_id = ?", employeeID)
	}

	if startDate != "" {
		query = query.Where("document_staffs.created_at >= ?", startDate)
	}

	if endDate != "" {
		query = query.Where("document_staffs.created_at <= ?", endDate)
	}

	var total int64
	query.Count(&total)

	type DocumentStaffResponse struct {
		ID           string    `json:"id"`
		UserID       *string   `json:"user_id,omitempty"`
		EmployeeID   *string   `json:"employee_id,omitempty"`
		FileURL      string    `json:"file_url"`
		Subject      string    `json:"subject"`
		FileName     string    `json:"file_name"`
		PublicID     string    `json:"public_id"`
		ResourceType string    `json:"resource_type"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		OwnerName    string    `json:"owner_name"`
	}

	var documents []DocumentStaffResponse

	if err := query.
		Order("document_staffs.created_at DESC").
		Limit(limitInt).
		Offset(offset).
		Scan(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data dokumen: " + err.Error(),
		})
		return
	}

	formattedDocuments := make([]map[string]interface{}, len(documents))
	for i, doc := range documents {
		formattedDoc := map[string]interface{}{
			"id":            doc.ID,
			"file_url":      doc.FileURL,
			"subject":       doc.Subject,
			"file_name":     doc.FileName,
			"public_id":     doc.PublicID,
			"resource_type": doc.ResourceType,
			"created_at":    doc.CreatedAt,
			"updated_at":    doc.UpdatedAt,
			"owner_name":    doc.OwnerName,
		}

		if doc.UserID != nil && *doc.UserID != "" {
			formattedDoc["user_id"] = *doc.UserID
			formattedDoc["employee_id"] = nil
		} else if doc.EmployeeID != nil && *doc.EmployeeID != "" {
			formattedDoc["employee_id"] = *doc.EmployeeID
			formattedDoc["user_id"] = nil
		} else {
			formattedDoc["user_id"] = nil
			formattedDoc["employee_id"] = nil
		}

		formattedDocuments[i] = formattedDoc
	}

	totalPages := int(math.Ceil(float64(total) / float64(limitInt)))

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data dokumen staff",
		"data": gin.H{
			"documents": formattedDocuments,
			"pagination": gin.H{
				"current_page": pageInt,
				"per_page":     limitInt,
				"total_items":  total,
				"total_pages":  totalPages,
			},
		},
	})
}

// ======================================================
// GET MY DOCUMENTS STAFF - FOR LOGGED IN USER
// ======================================================
func GetMyDocumentsStaff(c *gin.Context) {
	employeeIDRaw, employeeExists := c.Get("employeeID")
	if !employeeExists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized - employee identifier not found",
		})
		return
	}

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	subject := c.Query("subject")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 1 {
		limitInt = 10
	}

	if limitInt > 50 {
		limitInt = 50
	}

	offset := (pageInt - 1) * limitInt

	employeeID, ok := employeeIDRaw.(string)
	if !ok || employeeID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid employeeID"})
		return
	}

	var emp employee.Employee
	if err := database.DB.First(&emp, "id = ?", employeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee tidak ditemukan"})
		return
	}

	query := database.DB.Model(&DocumentStaff{}).
		Select(`document_staffs.id,
				document_staffs.employee_id,
				document_staffs.file_url,
				document_staffs.subject,
				document_staffs.file_name,
				document_staffs.public_id,
				document_staffs.resource_type,
				document_staffs.created_at,
				document_staffs.updated_at,
				employees.name as owner_name`).
		Joins("LEFT JOIN employees ON employees.id = document_staffs.employee_id").
		Where("document_staffs.employee_id = ?", employeeID)

	if subject != "" {
		query = query.Where("document_staffs.subject LIKE ?", "%"+subject+"%")
	}

	if startDate != "" {
		query = query.Where("document_staffs.created_at >= ?", startDate)
	}

	if endDate != "" {
		query = query.Where("document_staffs.created_at <= ?", endDate)
	}

	var total int64
	query.Count(&total)

	type MyDocumentResponse struct {
		ID           string    `json:"id"`
		EmployeeID   string    `json:"employee_id"`
		FileURL      string    `json:"file_url"`
		Subject      string    `json:"subject"`
		FileName     string    `json:"file_name"`
		PublicID     string    `json:"public_id"`
		ResourceType string    `json:"resource_type"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		OwnerName    string    `json:"owner_name"`
	}

	var documents []MyDocumentResponse

	if err := query.
		Order("document_staffs.created_at DESC").
		Limit(limitInt).
		Offset(offset).
		Scan(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data dokumen: " + err.Error(),
		})
		return
	}

	formattedDocuments := make([]map[string]interface{}, len(documents))
	for i, doc := range documents {
		formattedDoc := map[string]interface{}{
			"id":            doc.ID,
			"file_url":      doc.FileURL,
			"subject":       doc.Subject,
			"file_name":     doc.FileName,
			"public_id":     doc.PublicID,
			"resource_type": doc.ResourceType,
			"created_at":    doc.CreatedAt,
			"updated_at":    doc.UpdatedAt,
			"owner_name":    doc.OwnerName,
		}

		formattedDoc["employee_id"] = doc.EmployeeID
		formattedDocuments[i] = formattedDoc
	}

	totalPages := int(math.Ceil(float64(total) / float64(limitInt)))

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data dokumen Anda",
		"data": gin.H{
			"documents": formattedDocuments,
			"pagination": gin.H{
				"current_page": pageInt,
				"per_page":     limitInt,
				"total_items":  total,
				"total_pages":  totalPages,
			},
		},
	})
}

// ======================================================
// UPDATE FOR ADMIN ONLY
// ======================================================
func UpdateDocumentStaffAdmin(c *gin.Context) {
	documentID := c.Param("id")
	userID := c.PostForm("user_id")
	employeeID := c.PostForm("employee_id")
	subject := c.PostForm("subject")

	if subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject wajib diisi"})
		return
	}

	if userID == "" && employeeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UserID atau EmployeeID harus diisi"})
		return
	}

	var document DocumentStaff
	if err := database.DB.First(&document, "id = ?", documentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	if document.PublicID != "" {
		err := config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus dokumen lama: " + err.Error()})
			return
		}
	}

	fileHeader, err := c.FormFile("file")
	if err == nil {
		src, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
			return
		}
		defer src.Close()

		fileBytes, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
			return
		}
		reader := bytes.NewReader(fileBytes)

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		var resourceType, folder string
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			resourceType = "image"
			folder = "gambar"
		case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
			resourceType = "raw"
			folder = "document_staff"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
			return
		}

		uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
			return
		}

		document.FileName = fileHeader.Filename
		document.FileURL = uploadResult.SecureURL
		document.PublicID = uploadResult.PublicID
		document.ResourceType = resourceType
	}

	document.Subject = subject
	if userID != "" {
		document.UserID = userID
	}
	if employeeID != "" {
		document.EmployeeID = employeeID
	}

	if err := database.DB.Save(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui dokumen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Dokumen berhasil diperbarui",
		"document": document,
	})
}

// ======================================================
// UPDATE MY DOCUMENT - FOR LOGGED IN EMPLOYEE
// ======================================================
func UpdateMyDocumentStaff(c *gin.Context) {
	employeeIDRaw, exists := c.Get("employeeID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - employeeID not found"})
		return
	}

	employeeID, ok := employeeIDRaw.(string)
	if !ok || employeeID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid employeeID"})
		return
	}

	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID wajib diisi"})
		return
	}

	subject := c.PostForm("subject")
	if subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subject wajib diisi"})
		return
	}

	var document DocumentStaff
	if err := database.DB.First(&document, "id = ? AND employee_id = ?", documentID, employeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan atau Anda tidak memiliki akses"})
		return
	}

	updates := map[string]interface{}{
		"subject":     subject,
		"employee_id": employeeID,
		"updated_at":  time.Now(),
	}

	fileHeader, err := c.FormFile("file")
	if err == nil {
		if document.PublicID != "" {
			err := config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus dokumen lama: " + err.Error()})
				return
			}
		}

		src, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
			return
		}
		defer src.Close()

		fileBytes, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
			return
		}
		reader := bytes.NewReader(fileBytes)

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		var resourceType, folder string
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			resourceType = "image"
			folder = "gambar"
		case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
			resourceType = "raw"
			folder = "document_staff"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
			return
		}

		uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
			return
		}

		updates["file_name"] = fileHeader.Filename
		updates["file_url"] = uploadResult.SecureURL
		updates["public_id"] = uploadResult.PublicID
		updates["resource_type"] = resourceType
	}

	fieldsToUpdate := []string{"subject", "employee_id", "updated_at"}
	if fileHeader != nil {
		fieldsToUpdate = append(fieldsToUpdate, "file_name", "file_url", "public_id", "resource_type")
	}

	if err := database.DB.Model(&document).
		Select(fieldsToUpdate).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui dokumen: " + err.Error()})
		return
	}

	if err := database.DB.First(&document, "id = ?", documentID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data dokumen terbaru: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Dokumen berhasil diperbarui",
		"document": document,
	})
}

// ======================================================
// DELETE DOCUMENT STAFF - FOR ALL ROLES
// ======================================================
func DeleteDocumentStaff(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID wajib diisi"})
		return
	}

	roleRaw, exists := c.Get("role")
	var role string
	if exists {
		if r, ok := roleRaw.(string); ok {
			role = r
		}
	}

	employeeIDRaw, employeeExists := c.Get("employeeID")
	var employeeID string
	if employeeExists {
		if eID, ok := employeeIDRaw.(string); ok {
			employeeID = eID
		}
	}

	var document DocumentStaff
	query := database.DB.Where("id = ?", documentID)

	if role != "admin" && role != "superadmin" {
		if employeeID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - employeeID not found"})
			return
		}
		query = query.Where("employee_id = ?", employeeID)
	}

	if err := query.First(&document).Error; err != nil {
		if role == "admin" || role == "superadmin" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan atau Anda tidak memiliki akses"})
		}
		return
	}

	if document.PublicID != "" {
		if err := config.DeleteFromCloudinary(document.PublicID, document.ResourceType); err != nil {
			fmt.Printf("Warning: Failed to delete file from Cloudinary: %v\n", err)
		}
	}

	if err := database.DB.Delete(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus dokumen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Dokumen berhasil dihapus",
		"data": gin.H{
			"id":         document.ID,
			"file_name":  document.FileName,
			"subject":    document.Subject,
			"deleted_at": time.Now(),
		},
	})
}
