package mocks

import (
	"context"

	"Tages/internal/dto"
	pb "Tages/pkg"

	"google.golang.org/grpc"
)

// MockFileServiceServer паникает по умолчанию, можно переопределять методы
type MockFileServiceServer struct {
	UploadFileFn   func(ctx context.Context, req *pb.UploadRequest) (*pb.UploadResponse, error)
	DownloadFileFn func(req *pb.DownloadRequest, stream pb.FileService_DownloadFileServer) error
	ListFilesFn    func(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error)
}

func (m *MockFileServiceServer) UploadFile(ctx context.Context, req *pb.UploadRequest) (*pb.UploadResponse, error) {
	if m.UploadFileFn != nil {
		return m.UploadFileFn(ctx, req)
	}
	panic("UploadFile not implemented")
}

func (m *MockFileServiceServer) DownloadFile(req *pb.DownloadRequest, stream pb.FileService_DownloadFileServer) error {
	if m.DownloadFileFn != nil {
		return m.DownloadFileFn(req, stream)
	}
	panic("DownloadFile not implemented")
}

func (m *MockFileServiceServer) ListFiles(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	if m.ListFilesFn != nil {
		return m.ListFilesFn(ctx, req)
	}
	panic("ListFiles not implemented")
}

// func (m *MockFileServiceServer) mustEmbedUnimplementedFileServiceServer() {}

type MockFileServiceClient struct {
	UploadFileFn   func(ctx context.Context, in *pb.UploadRequest, opts ...grpc.CallOption) (*pb.UploadResponse, error)
	DownloadFileFn func(ctx context.Context, in *pb.DownloadRequest, opts ...grpc.CallOption) (pb.FileService_DownloadFileClient, error)
	ListFilesFn    func(ctx context.Context, in *pb.ListRequest, opts ...grpc.CallOption) (*pb.ListResponse, error)
}

func (m *MockFileServiceClient) UploadFile(ctx context.Context, in *pb.UploadRequest, opts ...grpc.CallOption) (*pb.UploadResponse, error) {
	if m.UploadFileFn != nil {
		return m.UploadFileFn(ctx, in, opts...)
	}
	panic("UploadFile not implemented")
}

func (m *MockFileServiceClient) DownloadFile(ctx context.Context, in *pb.DownloadRequest, opts ...grpc.CallOption) (pb.FileService_DownloadFileClient, error) {
	if m.DownloadFileFn != nil {
		return m.DownloadFileFn(ctx, in, opts...)
	}
	panic("DownloadFile not implemented")
}

func (m *MockFileServiceClient) ListFiles(ctx context.Context, in *pb.ListRequest, opts ...grpc.CallOption) (*pb.ListResponse, error) {
	if m.ListFilesFn != nil {
		return m.ListFilesFn(ctx, in, opts...)
	}
	panic("ListFiles not implemented")
}

type MockStorage struct {
	AddFileFn           func(ctx context.Context, f dto.File) error
	WithInTransactionFn func(ctx context.Context, fn func(txCtx context.Context) error) error
	GetAllFilesFn       func() ([]dto.File, error)
}

func (m *MockStorage) AddFile(ctx context.Context, f dto.File) error {
	return m.AddFileFn(ctx, f)
}

func (m *MockStorage) WithInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return m.WithInTransactionFn(ctx, fn)
}

func (m *MockStorage) GetAllFiles() ([]dto.File, error) {
	return m.GetAllFilesFn()
}
