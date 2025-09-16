package service

import (
	"Tages/internal/cache"
	"Tages/internal/dto"
	"Tages/internal/storage"
	pb "Tages/pkg"
	"context"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const maxFileSize = 100 * 1024 * 1024

type ServiceFile struct {
	pb.UnimplementedFileServiceServer
	uploadCh  chan struct{}
	listFiles chan struct{}
	uploadDir string
	cache     cache.CacheInterface
	logger    *logrus.Logger
	mu        sync.Mutex
	storage   storage.Storage
	fileMeta  map[string]dto.File
}

func NewServicefile(ctx context.Context, logger *logrus.Logger, cache cache.CacheInterface, storage storage.Storage) *ServiceFile {
	dir := viper.GetString("upload.dir")
	if !ensureDir(logger, dir) {
		return nil
	}

	return &ServiceFile{
		uploadCh:  make(chan struct{}, 10),
		listFiles: make(chan struct{}, 100),
		uploadDir: dir,
		cache:     cache,
		logger:    logger,
		fileMeta:  make(map[string]dto.File),
	}
}

func ensureDir(logger *logrus.Logger, dir string) bool {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		logger.Info("Directory not found, creating:", dir)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			logger.WithError(err).Error("Can't create directory")
			return false
		}
	} else if err != nil {
		logger.WithError(err).Error("Error checking directory")
		return false
	}
	return true
}
