package dto

import "time"

type File struct {
	Name      string `json:"name"`
	Path      string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
