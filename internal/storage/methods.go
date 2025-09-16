package storage

import (
	"Tages/internal/dto"
	"context"

	"github.com/pkg/errors"
)

var ErrNotFound = errors.New("not found")

func (s *Storage) AddFile(ctx context.Context, f dto.File) error {
	err := s.conn.WithContext(ctx).Create(f).Error
	if err != nil {
		return errors.Wrapf(err, "cant create file %v", f)
	}

	return nil
}

func (s *Storage) GetAllFiles() ([]dto.File, error) {
	var files []dto.File

	err := s.conn.Find(&files).Error
	if err != nil {
		return nil, errors.Wrap(err, "cant get files")
	}

	if len(files) == 0 {
		return nil, ErrNotFound
	}

	return files, nil
}
