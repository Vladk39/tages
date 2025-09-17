package tests

import (
	"Tages/internal/cache"
	"Tages/internal/service"
	pb "Tages/pkg"
	"Tages/pkg/mocks"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockDownloadStream struct {
	grpc.ServerStream
	sent []*pb.DownloadResponse
}

func (m *mockDownloadStream) Send(resp *pb.DownloadResponse) error {
	m.sent = append(m.sent, resp)
	return nil
}

func TestDownloadFileStream(t *testing.T) {
	dir := t.TempDir()
	viper.Set("upload.dir", dir)

	content := []byte("test data")
	path := filepath.Join(dir, "file.txt")
	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err)

	logger := logrus.New()
	c := cache.NewCache(logger, make(chan bool, 1))
	mockStorage := &mocks.MockStorage{}
	srv, _ := service.NewServicefile(context.Background(), logger, c, mockStorage)

	stream := &mockDownloadStream{}
	req := &pb.DownloadRequest{Filename: "file.txt"}

	err = srv.DownloadFileStream(req, stream)
	require.NoError(t, err)
	require.NotEmpty(t, stream.sent)
	require.Equal(t, content, bytes.Join(func() [][]byte {
		var chunks [][]byte
		for _, s := range stream.sent {
			chunks = append(chunks, s.Data)
		}
		return chunks
	}(), nil))
}
