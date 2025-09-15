package service

import (
	"Tages/internal/dto"
	"Tages/internal/helper"
	pb "Tages/pkg"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type UploadFileStream = pb.FileService_UploadFileServer

type ServiceFile struct {
	pb.UnimplementedFileServiceServer
	uploadCh    chan struct{}
	listFiles   chan struct{}
	uploadDir   string
	metaDataDir string
	logger      *logrus.Logger
	mu          sync.Mutex
	fileMeta    map[string]dto.File
}

func NewServicefile(logger *logrus.Logger, ctx context.Context) *ServiceFile {
	dir := viper.GetString("upload.dir")
	if !ensureDir(logger, dir) {
		return nil
	}

	return &ServiceFile{
		uploadCh:  make(chan struct{}, 10),
		listFiles: make(chan struct{}, 100),
		uploadDir: dir,

		logger:   logger,
		fileMeta: make(map[string]dto.File),
	}
}

func (s *ServiceFile) UploadFile(ctx context.Context, stream UploadFileStream) error {
	// Ограничение по параллельным аплоадам
	s.uploadCh <- struct{}{}
	defer func() { <-s.uploadCh }()

	var (
		filename string
		file     *os.File
	)

	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	for {
		// Читаем чанк из стрима
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to receive chunk")
		}

		// Определяем имя файла
		if filename == "" {
			filename = filepath.Base(req.Filename)
			if filename == "" {
				return errors.New("invalid filename")
			}
			uniqFileName := helper.UniqueFilename(filename)
			path := filepath.Join(s.uploadDir, uniqFileName)

			_, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if err != nil {
				return errors.Wrap(err, "failed to create file")
			}

		}

		// Пишем данные
		if _, err := file.Write(req.Data); err != nil {
			return errors.Wrap(err, "failed to write chunk to file")
		}

		// Проверка отмены контекста
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
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
