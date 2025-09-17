package service

import (
	"Tages/internal/cache"
	"Tages/internal/dto"
	"Tages/internal/helper"
	"Tages/internal/storage"
	pb "Tages/pkg"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func NewServicefile(ctx context.Context, logger *logrus.Logger, cache cache.CacheInterface, storage storage.StorageInterface) (*ServiceFile, error) {
	dir := viper.GetString("upload.dir")
	if !ensureDir(logger, dir) {
		return nil, nil
	}

	return &ServiceFile{
		uploadCh:    make(chan struct{}, 10),
		listFilesCh: make(chan struct{}, 100),
		uploadDir:   dir,
		cache:       cache,
		logger:      logger,
		storage:     storage,
	}, nil
}

// получение файла, запись на диск
func (s *ServiceFile) UploadFileUnary(ctx context.Context, req *pb.UploadRequest) (*pb.UploadResponse, error) {
	s.uploadCh <- struct{}{}
	s.logger.WithField("свободных слотов в семафоре", cap(s.uploadCh)-len(s.uploadCh)).Info("состояние семафора")
	defer func() {
		<-s.uploadCh
	}()

	filename := filepath.Base(req.Filename)
	if filename == "" {
		s.logger.Warn("UploadFileUnary: filename is empty, using 'unknown'")
		filename = "unknow"
	}
	uniqueName := helper.UniqueFilename(filename)
	path := filepath.Join(s.uploadDir, uniqueName)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		s.logger.WithError(err).WithField("file", file).Error("cant create file")
		return nil, status.Errorf(codes.Internal, "failed to save file")
	}

	defer file.Close()

	if _, err := file.Write(req.Data); err != nil {
		_ = os.Remove(path)
		s.logger.WithError(err).WithField("file", file).Error("failed to write file ")
		return nil, status.Errorf(codes.Internal, "failed to save file")
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
		return nil, status.Errorf(codes.Internal, "failed to save file")
	}

	s.cache.Set(f)

	// добавить имя файла, если анноу, юзер будет знать где сохранен его файл
	return &pb.UploadResponse{
		Status: true,
		Name:   filename,
		Path:   path,
	}, nil
}

// загрузка файла стрим
func (s *ServiceFile) UploadFileStream(stream pb.FileService_UploadFileStreamServer) error {
	s.uploadCh <- struct{}{}
	defer func() { <-s.uploadCh }()

	var filename string
	var file *os.File
	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			savedFile := dto.File{
				Name:      filename,
				Path:      filepath.Join(s.uploadDir, filename),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}

			if err := s.storage.WithInTransaction(stream.Context(), func(txCtx context.Context) error {
				if err := s.storage.AddFile(txCtx, savedFile); err != nil {
					s.logger.WithError(err).Errorf("cant add file %v to database", savedFile)
					return err
				}
				return nil
			}); err != nil {
				s.logger.WithError(err).Error("transaction failed")
				return status.Errorf(codes.Internal, "failed to save file")
			}

			s.cache.Set(savedFile)

			return stream.SendAndClose(&pb.UploadResponse{
				Status: true,
				Name:   filename,
				Path:   filepath.Join(s.uploadDir, filename),
			})
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive chunk")
		}

		if file == nil {
			filename = req.GetFilename()
			if filename == "" {
				return status.Error(codes.InvalidArgument, "filename is required")
			}
			uniqueName := helper.UniqueFilename(filename)
			path := filepath.Join(s.uploadDir, uniqueName)
			file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to save file")
			}
			filename = uniqueName
		}

		if _, err := file.Write(req.GetData()); err != nil {
			return status.Errorf(codes.Internal, "failed to save file")
		}
	}
}

// загрузка файлов
func (s *ServiceFile) DownloadFileStream(req *pb.DownloadRequest, stream pb.FileService_DownloadFileStreamServer) error {
	s.uploadCh <- struct{}{}
	s.logger.WithField("свободных слотов в семафоре", cap(s.uploadCh)-len(s.uploadCh)).Info("состояние семафора")
	defer func() {
		<-s.uploadCh
	}()

	filename := req.GetFilename()
	if filename == "" {
		return status.Error(codes.InvalidArgument, "file name is required")
	}

	path := filepath.Join(s.uploadDir, filename)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return status.Error(codes.NotFound, "file not found")
		}
		return status.Errorf(codes.Internal, "failed to save file")
	}
	defer file.Close()

	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return status.Errorf(codes.Internal, "failed to save file")
		}
		if n == 0 {
			break
		}

		resp := &pb.DownloadResponse{
			Data: buf[:n],
		}

		if err := stream.Send(resp); err != nil {
			return status.Errorf(codes.Internal, "failed to save file")
		}
	}
	return nil
}

func (s *ServiceFile) DownloadFileUnary(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error) {
	s.uploadCh <- struct{}{}
	s.logger.WithField("свободных слотов в семафоре", cap(s.uploadCh)-len(s.uploadCh)).Info("состояние семафора")
	defer func() {
		<-s.uploadCh
	}()

	filename := req.Filename
	if filename == "" {
		return nil, status.Error(codes.InvalidArgument, "file name is required")
	}

	path := filepath.Join(s.uploadDir, filename)

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		s.logger.WithError(err).WithField("path", path).Error("cannot open file")
		return nil, status.Error(codes.Internal, "failed to download file")
	}
	defer file.Close()

	data, err := os.ReadFile(filename)
	if err != nil {
		s.logger.WithError(err).WithField("file", path).Error("cannot read file")
		return nil, status.Error(codes.Internal, "failed to download file")
	}

	return &pb.DownloadResponse{
		Data: data,
	}, nil

}

func (s *ServiceFile) ListFiles(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
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

	return resp, nil
}

// прогрев кеша при старте
func (s *ServiceFile) HeatCache(ctx context.Context) error {
	s.logger.Info("starting to fill the cache")

	files, err := s.storage.GetAllFiles(ctx)
	if err != nil {
		s.logger.Error("error in heat cache")
		return err
	}

	s.cache.Warm(files)
	s.logger.Info("cache is full")
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
