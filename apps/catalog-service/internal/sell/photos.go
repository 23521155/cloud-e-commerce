package sell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const (
	MaxPhotos     = 5
	MaxPhotoBytes = 5 << 20
)

var (
	ErrTooManyPhotos  = errors.New("too many photos")
	ErrPhotoTooLarge  = errors.New("photo is too large")
	ErrNotAnImage     = errors.New("photo is not a supported image")
	ErrPhotosDisabled = errors.New("photo uploads are not available")
	ErrPhotoNotFound  = errors.New("photo not found")
)

var photoExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type PhotoStore interface {
	Put(ctx context.Context, name, contentType string, body io.ReadSeekCloser, metadata map[string]string) error
	Delete(ctx context.Context, name string) error
	Open(ctx context.Context, name string) (io.ReadCloser, string, int64, error)
}

type Upload struct {
	ContentType string
	Size        int64
	open        func() (multipart.File, error)
}

func Uploads(files []*multipart.FileHeader) ([]Upload, error) {
	if len(files) > MaxPhotos {
		return nil, ErrTooManyPhotos
	}

	uploads := make([]Upload, 0, len(files))
	for _, fh := range files {
		if fh.Size > MaxPhotoBytes {
			return nil, ErrPhotoTooLarge
		}
		if fh.Size == 0 {
			continue
		}

		f, err := fh.Open()
		if err != nil {
			return nil, err
		}
		head := make([]byte, 512)
		n, _ := io.ReadFull(f, head)
		f.Close()

		contentType := http.DetectContentType(head[:n])
		if _, ok := photoExtensions[contentType]; !ok {
			return nil, ErrNotAnImage
		}
		uploads = append(uploads, Upload{ContentType: contentType, Size: fh.Size, open: fh.Open})
	}
	return uploads, nil
}

func photoName(requestID int64, position int, contentType string) string {
	return fmt.Sprintf("%d/%d%s", requestID, position, photoExtensions[contentType])
}

type Photo struct {
	BlobName    string     `db:"blob_name" json:"-"`
	RequestID   int64      `db:"request_id" json:"-"`
	Position    int        `db:"position" json:"position"`
	ContentType string     `db:"content_type" json:"contentType"`
	SizeBytes   int64      `db:"size_bytes" json:"sizeBytes"`
	Width       *int       `db:"width" json:"width"`
	Height      *int       `db:"height" json:"height"`
	ProcessedAt *time.Time `db:"processed_at" json:"processedAt"`
}
