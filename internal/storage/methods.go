package storage

import (
	"Tages/internal/dto"
	"Tages/internal/metrics"
	"context"
	"time"

	"github.com/pkg/errors"
)

var ErrNotFound = errors.New("not found")

func (s *Storage) AddFile(ctx context.Context, f dto.File) error {
	start := time.Now()

	err := s.conn.WithContext(ctx).Create(f).Error

	status := "success"
	if err != nil {
		status = "error"
	}

	metrics.DBMetricsFunc(status, "add_file", start)
	return err
}

func (s *Storage) GetAllFiles(ctx context.Context) ([]dto.File, error) {
	start := time.Now()

	var files []dto.File
	err := s.conn.WithContext(ctx).Find(&files).Error

	status := "success"
	if err != nil {
		status = "error"
	} else if len(files) == 0 {
		err = ErrNotFound
		status = "not_found"
	}

	metrics.DBMetricsFunc(status, "get_all_files", start)
	return files, err
}
