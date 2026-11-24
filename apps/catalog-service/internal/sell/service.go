package sell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

var ErrInvalidInput = errors.New("invalid sell request")

// store allows service tests to use a fake repository.
type store interface {
	Create(ctx context.Context, in CreateInput) (Request, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit int) ([]Request, error)
	Photo(ctx context.Context, requestID int64, position int) (Photo, error)
}

type Service struct {
	store  store
	photos PhotoStore
}

func NewService(store store, photos PhotoStore) *Service {
	return &Service{store: store, photos: photos}
}

const listLimit = 100

func (s *Service) List(ctx context.Context) ([]Request, error) {
	return s.store.List(ctx, listLimit)
}

func (s *Service) OpenPhoto(ctx context.Context, requestID int64, position int) (io.ReadCloser, string, int64, error) {
	if s.photos == nil {
		return nil, "", 0, ErrPhotoNotFound
	}
	photo, err := s.store.Photo(ctx, requestID, position)
	if err != nil {
		return nil, "", 0, err
	}
	return s.photos.Open(ctx, photo.BlobName)
}

func (s *Service) Create(ctx context.Context, in CreateInput, uploads []Upload) (Request, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Author = strings.TrimSpace(in.Author)
	in.Edition = strings.TrimSpace(in.Edition)
	in.Binding = strings.TrimSpace(in.Binding)
	in.Grade = strings.TrimSpace(in.Grade)
	in.Faults = strings.TrimSpace(in.Faults)
	in.Virtues = strings.TrimSpace(in.Virtues)
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)

	fields := []struct {
		name     string
		value    string
		max      int
		required bool
	}{
		{"title", in.Title, 600, true},
		{"author", in.Author, 150, true},
		{"edition", in.Edition, 200, false},
		{"faults", in.Faults, 4000, true},
		{"virtues", in.Virtues, 4000, false},
		{"name", in.Name, 150, true},
		{"email", in.Email, 254, true},
		{"phone", in.Phone, 30, false},
	}

	for _, field := range fields {
		if field.required && field.value == "" {
			return Request{}, invalid(field.name)
		}

		// SQL Server NVARCHAR limits count UTF-16 code units.
		if len(utf16.Encode([]rune(field.value))) > field.max {
			return Request{}, invalid(field.name)
		}
	}

	switch in.Binding {
	case "Hardcover", "Paperback", "Other":
	default:
		return Request{}, invalid("binding")
	}

	switch in.Grade {
	case "Fine", "Very good", "Good", "Fair", "Poor", "Not sure":
	default:
		return Request{}, invalid("grade")
	}

	address, err := mail.ParseAddress(in.Email)
	if err != nil {
		return Request{}, invalid("email")
	}
	if address.Address != in.Email {
		return Request{}, invalid("email")
	}

	if in.Year != nil && (*in.Year < 1 || *in.Year > time.Now().Year()) {
		return Request{}, invalid("year")
	}

	if in.Asking != nil && *in.Asking <= 0 {
		return Request{}, invalid("asking")
	}

	if len(uploads) > 0 && s.photos == nil {
		return Request{}, ErrPhotosDisabled
	}

	request, err := s.store.Create(ctx, in)
	if err != nil {
		return Request{}, err
	}

	if err := s.storePhotos(ctx, request.ID, uploads); err != nil {
		if delErr := s.store.Delete(context.WithoutCancel(ctx), request.ID); delErr != nil {
			err = errors.Join(err, delErr)
		}
		return Request{}, err
	}

	return request, nil
}

func (s *Service) storePhotos(ctx context.Context, requestID int64, uploads []Upload) error {
	stored := make([]string, 0, len(uploads))
	for i, u := range uploads {
		position := i + 1
		name := photoName(requestID, position, u.ContentType)

		f, err := u.open()
		if err == nil {
			err = s.photos.Put(ctx, name, u.ContentType, f, map[string]string{
				"requestid": strconv.FormatInt(requestID, 10),
				"position":  strconv.Itoa(position),
			})
			f.Close()
		}
		if err != nil {
			for _, done := range stored {
				_ = s.photos.Delete(context.WithoutCancel(ctx), done)
			}
			return fmt.Errorf("store photo %s: %w", name, err)
		}
		stored = append(stored, name)
	}
	return nil
}

func invalid(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, field)
}