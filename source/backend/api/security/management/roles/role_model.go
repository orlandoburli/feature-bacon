package roles

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Role struct {
	ID        uuid.UUID      `json:"id" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	Name      string         `json:"name" binding:"required"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index"`
}

type CreateRoleRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateRoleRequest struct {
	Name string `json:"name" binding:"required"`
}
