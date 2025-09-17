package tests

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"Tages/internal/cache"
	"Tages/internal/dto"
	"Tages/internal/service"
	pb "Tages/pkg"
	"Tages/pkg/mocks"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockUploadStream struct {
	grpc.ServerStream
	reqs []*pb.UploadRequest
	resp *pb.UploadResponse
	i    int
}

func (m *mockUploadStream) Recv() (*pb.UploadRequest, error) {
	if m.i >= len(m.reqs) {
		return nil, io.EOF
	}
	r := m.reqs[m.i]
	m.i++
	return r, nil
}

func (m *mockUploadStream) SendAndClose(resp *pb.UploadResponse) error {
	m.resp = resp
	return nil
}
func (m *mockUploadStream) Context() context.Context {
	return context.Background()
}

func TestUploadFileStream(t *testing.T) {
	dir := t.TempDir()
	viper.Set("upload.dir", dir)

	logger := logrus.New()
	c := cache.NewCache(logger, make(chan bool, 1))
	mockStorage := &mocks.MockStorage{
		AddFileFn: func(ctx context.Context, f dto.File) error { return nil },
		WithInTransactionFn: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}
	srv, _ := service.NewServicefile(context.Background(), logger, c, mockStorage)

	data := []byte("tetesttesttesttesttesttetesttesttesttesttesttesttesttestteststtetesttesttesttesttesttesttesttestteststtetesttesttesttesttesttesttesttestteststtetesttesttesttesttesttesttesttestteststtetesttesttesttesttesttesttesttestteststtesttesttesttestst")
	stream := &mockUploadStream{
		reqs: []*pb.UploadRequest{
			{Filename: "file.txt", Data: data},
		},
	}

	err := srv.UploadFileStream(stream)
	require.NoError(t, err)
	require.NotNil(t, stream.resp)
	require.True(t, stream.resp.Status)

	_, err = os.Stat(filepath.Join(dir, stream.resp.Name))
	require.NoError(t, err)
}
