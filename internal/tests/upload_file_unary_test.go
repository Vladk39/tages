package tests

import (
	"Tages/internal/cache"
	"Tages/internal/dto"
	"Tages/internal/service"
	pb "Tages/pkg"
	"Tages/pkg/mocks"
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestUploadFileUnary(t *testing.T) {
	dir := t.TempDir()
	viper.Set("upload.dir", dir)

	logger := logrus.New()

	c := cache.NewCache(logger, make(chan bool, 1))

	mockStorage := &mocks.MockStorage{
		AddFileFn: func(ctx context.Context, f dto.File) error { return nil },
		WithInTransactionFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	srv, _ := service.NewServicefile(context.Background(), logger, c, mockStorage)

	imgData, _ := os.ReadFile("./cat.jpg")

	req := &pb.UploadRequest{
		Filename: "itsCat",
		Data:     imgData,
	}

	resp, err := srv.UploadFileUnary(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Status)

	_, err = os.Stat(resp.Path)
	require.NoError(t, err)
}

func TestUploadFileUnaryConcurrent(t *testing.T) {
	numGoroutines := 25
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	logger := logrus.New()
	c := cache.NewCache(logger, make(chan bool, 1))

	mockStorage := &mocks.MockStorage{
		AddFileFn: func(ctx context.Context, f dto.File) error { return nil },
		WithInTransactionFn: func(ctx context.Context, fn func(txCtx context.Context) error) error {
			return fn(ctx)
		},
	}

	imgData, err := os.ReadFile("./cat.jpg")
	require.NoError(t, err)

	dir := t.TempDir()
	viper.Set("upload.dir", dir)

	srv := service.NewServicefile(context.Background(), logger, c, mockStorage)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()

			req := &pb.UploadRequest{
				Filename: fmt.Sprintf("itsCat_%d", i+1),
				Data:     imgData,
			}

			resp, err := srv.UploadFileUnary(context.Background(), req)

			require.NoError(t, err)
			require.True(t, resp.Status)

			_, err = os.Stat(resp.Path)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()
}
