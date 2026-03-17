package implementations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/user/protocol_registry/internal/entities"
)

type ProtocolStorageS3 struct {
	client *s3.Client
	bucket string
}

func NewProtocolStorageS3(client *s3.Client, bucket string) *ProtocolStorageS3 {
	return &ProtocolStorageS3{
		client: client,
		bucket: bucket,
	}
}

type manifest struct {
	EntryPoint string   `json:"entry_point"`
	Files      []string `json:"files"`
}

// UploadFileSet uploads all files and then writes a manifest.
func (s *ProtocolStorageS3) UploadFileSet(ctx context.Context, serviceName string, protocolType entities.ProtocolType, fileSet entities.ProtoFileSet) error {
	prefix := protocolPrefix(serviceName, protocolType)
	for _, f := range fileSet.Files {
		key := prefix + "files/" + f.Path
		if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(f.Content),
			ContentType: aws.String("text/plain"),
		}); err != nil {
			return fmt.Errorf("upload file %q: %w", f.Path, err)
		}
	}

	return s.writeManifest(ctx, prefix, fileSet)
}

// DownloadFileSet reads the manifest and downloads all referenced files.
func (s *ProtocolStorageS3) DownloadFileSet(ctx context.Context, serviceName string, protocolType entities.ProtocolType) (entities.ProtoFileSet, error) {
	prefix := protocolPrefix(serviceName, protocolType)
	return s.readFileSet(ctx, prefix)
}

// UploadConsumerFileSet uploads consumer files and manifest.
func (s *ProtocolStorageS3) UploadConsumerFileSet(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType, fileSet entities.ProtoFileSet) error {
	prefix := consumerPrefix(consumerName, serverName, protocolType)
	for _, f := range fileSet.Files {
		key := prefix + "files/" + f.Path
		if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(f.Content),
			ContentType: aws.String("text/plain"),
		}); err != nil {
			return fmt.Errorf("upload consumer file %q: %w", f.Path, err)
		}
	}

	return s.writeManifest(ctx, prefix, fileSet)
}

// DownloadConsumerFileSet reads the consumer manifest and downloads all referenced files.
func (s *ProtocolStorageS3) DownloadConsumerFileSet(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType) (entities.ProtoFileSet, error) {
	prefix := consumerPrefix(consumerName, serverName, protocolType)
	return s.readFileSet(ctx, prefix)
}

// DeleteConsumer deletes all objects under the consumer prefix.
func (s *ProtocolStorageS3) DeleteConsumer(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType) error {
	prefix := consumerPrefix(consumerName, serverName, protocolType)

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list objects for delete: %w", err)
		}
		if len(page.Contents) == 0 {
			continue
		}

		objects := make([]types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			objects[i] = types.ObjectIdentifier{Key: obj.Key}
		}

		if _, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		}); err != nil {
			return fmt.Errorf("batch delete: %w", err)
		}
	}

	return nil
}

func (s *ProtocolStorageS3) writeManifest(ctx context.Context, prefix string, fileSet entities.ProtoFileSet) error {
	paths := make([]string, len(fileSet.Files))
	for i, f := range fileSet.Files {
		paths[i] = f.Path
	}

	data, err := json.Marshal(manifest{
		EntryPoint: fileSet.EntryPoint,
		Files:      paths,
	})
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	key := prefix + "manifest.json"
	if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	}); err != nil {
		return fmt.Errorf("upload manifest: %w", err)
	}

	return nil
}

func (s *ProtocolStorageS3) readFileSet(ctx context.Context, prefix string) (entities.ProtoFileSet, error) {
	manifestKey := prefix + "manifest.json"
	mOut, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(manifestKey),
	})
	if err != nil {
		return entities.ProtoFileSet{}, fmt.Errorf("download manifest: %w", err)
	}
	defer mOut.Body.Close()

	mData, err := io.ReadAll(mOut.Body)
	if err != nil {
		return entities.ProtoFileSet{}, fmt.Errorf("read manifest: %w", err)
	}

	var m manifest
	if err := json.Unmarshal(mData, &m); err != nil {
		return entities.ProtoFileSet{}, fmt.Errorf("unmarshal manifest: %w", err)
	}

	files := make([]entities.ProtoFile, len(m.Files))
	for i, path := range m.Files {
		key := prefix + "files/" + path
		fOut, err := s.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return entities.ProtoFileSet{}, fmt.Errorf("download file %q: %w", path, err)
		}

		content, err := io.ReadAll(fOut.Body)
		fOut.Body.Close()
		if err != nil {
			return entities.ProtoFileSet{}, fmt.Errorf("read file %q: %w", path, err)
		}

		files[i] = entities.ProtoFile{Path: path, Content: content}
	}

	return entities.ProtoFileSet{
		EntryPoint: m.EntryPoint,
		Files:      files,
	}, nil
}

func protocolPrefix(serviceName string, protocolType entities.ProtocolType) string {
	return fmt.Sprintf("protocols/%s/%s/", serviceName, protocolType.String())
}

func consumerPrefix(consumerName, serverName string, protocolType entities.ProtocolType) string {
	return fmt.Sprintf("consumers/%s/%s/%s/", serverName, consumerName, protocolType.String())
}
