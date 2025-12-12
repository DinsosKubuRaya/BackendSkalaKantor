package document_staff

import (
	"fmt"
	"time"

	"BackendKantorDinsos/domain/employee"
	"BackendKantorDinsos/domain/user"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentStaff struct {
	ID           string            `gorm:"type:char(36);primaryKey" json:"id"`
	UserID       string            `gorm:"type:char(36);null;default:null" json:"user_id"`
	EmployeeID   string            `gorm:"type:char(36);null;default:null" json:"employee_id"`
	FileURL      string            `gorm:"type:text" json:"file_url"`
	User         user.User         `gorm:"foreignKey:UserID;references:ID" json:"user"`
	Employee     employee.Employee `gorm:"foreignKey:EmployeeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"employee,omitempty"`
	Subject      string            `gorm:"type:varchar(255)" json:"subject"`
	FileName     string            `gorm:"type:varchar(500)" json:"file_name"`
	PublicID     string            `gorm:"type:varchar(255)" json:"public_id"`
	ResourceType string            `gorm:"type:varchar(20)" json:"resource_type"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func (d *DocumentStaff) BeforeCreate(tx *gorm.DB) (err error) {
	d.ID = uuid.NewString()
	return
}

func (d *DocumentStaff) BeforeSave(tx *gorm.DB) (err error) {
	if d.UserID == "" && d.EmployeeID == "" {
		return fmt.Errorf("either UserID or EmployeeID must be provided")
	}
	return
}
