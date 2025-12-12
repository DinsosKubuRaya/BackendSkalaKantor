package login

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID         string    `gorm:"primaryKey" json:"-"`
	EmployeeID string    `gorm:"type:char(36);index" json:"employee_id"`
	TokenHash  string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	ExpiresAt  time.Time `gorm:"index" json:"expires_at"`
	Revoked    bool      `gorm:"default:false" json:"revoked"`
	CreatedAt  time.Time `json:"created_at"`
	UserAgent  string    `gorm:"type:text" json:"user_agent"`
	IP         string    `gorm:"type:varchar(45)" json:"ip"`
}

func (e *RefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
	e.ID = uuid.NewString()
	return
}
