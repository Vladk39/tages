package dto

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID        uuid.UUID
	Name      string `json:"name"`
	Path      string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
