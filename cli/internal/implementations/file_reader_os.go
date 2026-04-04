package implementations

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/protocol-registry-cli/internal/usecases/publish_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/register_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/validate_protocol"
)

type rawProtoFile struct {
	path    string
	content []byte
}

func collectProtoFiles(dir string) ([]rawProtoFile, error) {
	var files []rawProtoFile
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, rawProtoFile{
			path:    rel,
			content: content,
		})
		return nil
	})
	return files, err
}

// PublishFileReader satisfies publish_protocol.FileReader.
type PublishFileReader struct{}

func NewPublishFileReader() *PublishFileReader {
	return &PublishFileReader{}
}

func (r *PublishFileReader) ReadProtoFiles(dir string) ([]publish_protocol.ProtoFile, error) {
	raw, err := collectProtoFiles(dir)
	if err != nil {
		return nil, err
	}
	result := make([]publish_protocol.ProtoFile, len(raw))
	for i, f := range raw {
		result[i] = publish_protocol.ProtoFile{Path: f.path, Content: f.content}
	}
	return result, nil
}

// RegisterFileReader satisfies register_consumer.FileReader.
type RegisterFileReader struct{}

func NewRegisterFileReader() *RegisterFileReader {
	return &RegisterFileReader{}
}

func (r *RegisterFileReader) ReadProtoFiles(dir string) ([]register_consumer.ProtoFile, error) {
	raw, err := collectProtoFiles(dir)
	if err != nil {
		return nil, err
	}
	result := make([]register_consumer.ProtoFile, len(raw))
	for i, f := range raw {
		result[i] = register_consumer.ProtoFile{Path: f.path, Content: f.content}
	}
	return result, nil
}

// ValidateFileReader satisfies validate_protocol.FileReader.
type ValidateFileReader struct{}

func NewValidateFileReader() *ValidateFileReader {
	return &ValidateFileReader{}
}

func (r *ValidateFileReader) ReadProtoFiles(dir string) ([]validate_protocol.ProtoFile, error) {
	raw, err := collectProtoFiles(dir)
	if err != nil {
		return nil, err
	}
	result := make([]validate_protocol.ProtoFile, len(raw))
	for i, f := range raw {
		result[i] = validate_protocol.ProtoFile{Path: f.path, Content: f.content}
	}
	return result, nil
}
