package mocks

import (
	"context"

	"Tages/internal/dto"
	pb "Tages/pkg"

	"google.golang.org/grpc"
)

type MockFileServiceServer struct {
	UploadFileStreamFn   func(stream pb.FileService_UploadFileStreamServer) error
	UploadFileUnaryFn    func(ctx context.Context, req *pb.UploadRequest) (*pb.UploadResponse, error)
	DownloadFileStreamFn func(req *pb.DownloadRequest, stream pb.FileService_DownloadFileStreamServer) error
	DownloadFileUnaryFn  func(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error)
	ListFilesFn          func(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error)
}

func (m *MockFileServiceServer) UploadFileStream(stream pb.FileService_UploadFileStreamServer) error {
	if m.UploadFileStreamFn != nil {
		return m.UploadFileStreamFn(stream)
	}
	panic("UploadFileStream not implemented")
}

func (m *MockFileServiceServer) UploadFileUnary(ctx context.Context, req *pb.UploadRequest) (*pb.UploadResponse, error) {
	if m.UploadFileUnaryFn != nil {
		return m.UploadFileUnaryFn(ctx, req)
	}
	panic("UploadFileUnary not implemented")
}

func (m *MockFileServiceServer) DownloadFileStream(req *pb.DownloadRequest, stream pb.FileService_DownloadFileStreamServer) error {
	if m.DownloadFileStreamFn != nil {
		return m.DownloadFileStreamFn(req, stream)
	}
	panic("DownloadFileStream not implemented")
}

func (m *MockFileServiceServer) DownloadFileUnary(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error) {
	if m.DownloadFileUnaryFn != nil {
		return m.DownloadFileUnaryFn(ctx, req)
	}
	panic("DownloadFileUnary not implemented")
}

func (m *MockFileServiceServer) ListFiles(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	if m.ListFilesFn != nil {
		return m.ListFilesFn(ctx, req)
	}
	panic("ListFiles not implemented")
}

// func (m *MockFileServiceServer) mustEmbedUnimplementedFileServiceServer() {}

type MockFileServiceClient struct {
	UploadFileStreamFn   func(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[pb.UploadRequest, pb.UploadResponse], error)
	UploadFileUnaryFn    func(ctx context.Context, in *pb.UploadRequest, opts ...grpc.CallOption) (*pb.UploadResponse, error)
	DownloadFileStreamFn func(ctx context.Context, in *pb.DownloadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.DownloadResponse], error)
	DownloadFileUnaryFn  func(ctx context.Context, in *pb.DownloadRequest, opts ...grpc.CallOption) (*pb.DownloadResponse, error)
	ListFilesFn          func(ctx context.Context, in *pb.ListRequest, opts ...grpc.CallOption) (*pb.ListResponse, error)
}

func (m *MockFileServiceClient) UploadFileStream(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[pb.UploadRequest, pb.UploadResponse], error) {
	if m.UploadFileStreamFn != nil {
		return m.UploadFileStreamFn(ctx, opts...)
	}
	panic("UploadFileStream not implemented")
}

func (m *MockFileServiceClient) UploadFileUnary(ctx context.Context, in *pb.UploadRequest, opts ...grpc.CallOption) (*pb.UploadResponse, error) {
	if m.UploadFileUnaryFn != nil {
		return m.UploadFileUnaryFn(ctx, in, opts...)
	}
	panic("UploadFileUnary not implemented")
}

func (m *MockFileServiceClient) DownloadFileStream(ctx context.Context, in *pb.DownloadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.DownloadResponse], error) {
	if m.DownloadFileStreamFn != nil {
		return m.DownloadFileStreamFn(ctx, in, opts...)
	}
	panic("DownloadFileStream not implemented")
}

func (m *MockFileServiceClient) DownloadFileUnary(ctx context.Context, in *pb.DownloadRequest, opts ...grpc.CallOption) (*pb.DownloadResponse, error) {
	if m.DownloadFileUnaryFn != nil {
		return m.DownloadFileUnaryFn(ctx, in, opts...)
	}
	panic("DownloadFileUnary not implemented")
}

func (m *MockFileServiceClient) ListFiles(ctx context.Context, in *pb.ListRequest, opts ...grpc.CallOption) (*pb.ListResponse, error) {
	if m.ListFilesFn != nil {
		return m.ListFilesFn(ctx, in, opts...)
	}
	panic("ListFiles not implemented")
}

type MockStorage struct {
	AddFileFn           func(ctx context.Context, f dto.File) error
	GetAllFilesFn       func(ctx context.Context) ([]dto.File, error)
	WithInTransactionFn func(ctx context.Context, tFunc func(ctx context.Context) error) error
}

func (m *MockStorage) AddFile(ctx context.Context, f dto.File) error {
	if m.AddFileFn != nil {
		return m.AddFileFn(ctx, f)
	}
	return nil
}

func (m *MockStorage) GetAllFiles(ctx context.Context) ([]dto.File, error) {
	if m.GetAllFilesFn != nil {
		return m.GetAllFilesFn(ctx)
	}
	return []dto.File{}, nil
}

func (m *MockStorage) WithInTransaction(ctx context.Context, tFunc func(ctx context.Context) error) error {
	if m.WithInTransactionFn != nil {
		return m.WithInTransactionFn(ctx, tFunc)
	}
	return tFunc(ctx)
}
