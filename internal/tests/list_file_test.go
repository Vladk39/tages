package tests

import (
	"Tages/internal/cache"
	"Tages/internal/dto"
	"Tages/internal/service"
	"Tages/pkg/mocks"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestHeatCache(t *testing.T) {
	logger := logrus.New()
	c := cache.NewCache(logger, make(chan bool, 1))

	files := []dto.File{
		{Name: "file1", Path: "/tmp/file1"},
		{Name: "file2", Path: "/tmp/file2"},
	}

	mockStorage := &mocks.MockStorage{
		GetAllFilesFn: func(ctx context.Context) ([]dto.File, error) {
			return files, nil
		},
	}

	srv, _ := service.NewServicefile(context.Background(), logger, c, mockStorage)

	err := srv.HeatCache(context.Background())
	require.NoError(t, err)

	got := c.GetFilesFromCache()
	require.Len(t, got, 2)
}
