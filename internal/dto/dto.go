package dto

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name      string    `json:"name"`
	Path      string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
