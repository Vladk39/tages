package service

import (
	"Tages/internal/cache"
	"Tages/internal/dto"
	"Tages/internal/helper"
	"Tages/internal/storage"
	pb "Tages/pkg"
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxFileSize = 100 * 1024 * 1024

type ServiceFile struct {
	pb.UnimplementedFileServiceServer
	uploadDir   string
	cache       cache.CacheInterface
	logger      *logrus.Logger
	storage     storage.StorageInterface
	uploadCh    chan struct{}
	listFilesCh chan struct{}
	mu          sync.Mutex
}

func NewServicefile(ctx context.Context, logger *logrus.Logger, cache cache.CacheInterface, storage storage.StorageInterface) *ServiceFile {
	dir := viper.GetString("upload.dir")
	if !ensureDir(logger, dir) {
		return nil
	}

	return &ServiceFile{
		uploadCh:    make(chan struct{}, 10),
		listFilesCh: make(chan struct{}, 100),
		uploadDir:   dir,
		cache:       cache,
		logger:      logger,
		storage:     storage,
	}
}

// получение файла, запись на диск
func (s *ServiceFile) UploadFileUnary(ctx context.Context, req *pb.UploadRequest) (*pb.UploadResponse, error) {
	s.uploadCh <- struct{}{}
	s.logger.WithField("свободных слотов в семафоре", cap(s.uploadCh)-len(s.uploadCh)).Info("состояние семафора")
	time.Sleep(500 * time.Millisecond)
	defer func() {
		<-s.uploadCh
	}()

	filename := filepath.Base(req.Filename)
	if filename == "" {
		filename = "unknow"
	}
	uniqueName := helper.UniqueFilename(filename)
	path := filepath.Join(s.uploadDir, uniqueName)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		s.logger.WithError(err).WithField("file", file).Error("cant create file")
		return nil, err
	}

	defer file.Close()

	if _, err := file.Write(req.Data); err != nil {
		_ = os.Remove(path)
		s.logger.WithError(err).WithField("file", file).Error("failed to write file ")
		return nil, err
	}

	f := dto.File{
		Name:      uniqueName,
		Path:      path,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.storage.WithInTransaction(ctx, func(txCtx context.Context) error {
		if err := s.storage.AddFile(txCtx, f); err != nil {
			s.logger.WithError(err).Errorf("cant add file %v to database", f)
			return err
		}
		return nil
	}); err != nil {
		s.logger.WithError(err).Error("transaction failed")
		return nil, err
	}

	s.cache.Set(f)

	// добавить имя файла, если анноу, юзер будет знать где сохранен его файл
	return &pb.UploadResponse{
		Status: true,
		Path:   path,
	}, nil
}

// func (s *ServiceFile) DownloadFile(ctx context.Context)

func (s *ServiceFile) ListFiles(ctx context.Context, req *pb.ListRequest) *pb.ListResponse {
	s.listFilesCh <- struct{}{}
	defer func() {
		<-s.listFilesCh
	}()

	files := s.cache.GetFilesFromCache()

	resp := &pb.ListResponse{}
	for _, v := range files {
		resp.Files = append(resp.Files, &pb.FileInfo{
			Name:      v.Name,
			Path:      v.Path,
			CreatedAt: timestamppb.New(v.CreatedAt),
			UpdatedAt: timestamppb.New(v.UpdatedAt),
		})
	}

	return resp
}

// прогрев кеша при старте
func (s *ServiceFile) HeatCache() error {
	files := []dto.File{}

	files, err := s.storage.GetAllFiles()
	if err != nil {
		s.logger.WithError(err).Error("failed warm up the cache")
		return err
	}

	s.cache.Warm(files)

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
