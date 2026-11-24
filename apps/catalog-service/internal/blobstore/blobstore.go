package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

type Store struct {
	container *container.Client
}

func New(ctx context.Context, connectionString, containerName string) (*Store, error) {
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("blob client: %w", err)
	}

	c := client.ServiceClient().NewContainerClient(containerName)
	if _, err := c.Create(ctx, nil); err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
		return nil, fmt.Errorf("create container %s: %w", containerName, err)
	}

	return &Store{container: c}, nil
}

func (s *Store) Put(ctx context.Context, name, contentType string, body io.ReadSeekCloser, metadata map[string]string) error {
	meta := make(map[string]*string, len(metadata))
	for k, v := range metadata {
		meta[k] = to.Ptr(v)
	}

	_, err := s.container.NewBlockBlobClient(name).Upload(ctx, body, &blockblob.UploadOptions{
		HTTPHeaders: &blob.HTTPHeaders{BlobContentType: to.Ptr(contentType)},
		Metadata:    meta,
	})
	return err
}

func (s *Store) Delete(ctx context.Context, name string) error {
	_, err := s.container.NewBlobClient(name).Delete(ctx, nil)
	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return nil
	}
	return err
}

var ErrNotFound = errors.New("blob not found")

func (s *Store) Open(ctx context.Context, name string) (io.ReadCloser, string, int64, error) {
	res, err := s.container.NewBlobClient(name).DownloadStream(ctx, nil)
	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return nil, "", 0, ErrNotFound
	}
	if err != nil {
		return nil, "", 0, err
	}

	contentType := "application/octet-stream"
	if res.ContentType != nil {
		contentType = *res.ContentType
	}
	var size int64 = -1
	if res.ContentLength != nil {
		size = *res.ContentLength
	}
	return res.Body, contentType, size, nil
}
