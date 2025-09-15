package service

import (
	"Tages/internal/dto"
	pb "Tages/pkg"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ServiceFile) SaveMetaData(file dto.File) error {
	data, err := json.Marshal(file)
	if err != nil {
		return errors.Wrap(err, "cant marshal metadata")
	}
	return os.WriteFile(file.Path, data, 0644)
}

func (s *ServiceFile) ListMetadata() ([]*pb.FileInfo, error) {
	entries, err := os.ReadDir(s.metaDataDir)
	if err != nil {
		return nil, err
	}

	var files []*pb.FileInfo

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".meta") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.uploadDir, entry.Name()))
		if err != nil {
			continue
		}
		var meta dto.File
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		files = append(files, &pb.FileInfo{
			Name:      meta.Name,
			CreatedAt: timestamppb.New(meta.CreatedAt),
			UpdatedAt: timestamppb.New(meta.UpdatedAt),
		})
	}
	return files, nil
}
