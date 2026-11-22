package sell

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf16"
)

var ErrInvalidInput = errors.New("invalid sell request")

// store allows service tests to use a fake repository.
type store interface {
	Create(ctx context.Context, in CreateInput) (Request, error)
}

type Service struct {
	store store
}

func NewService(store store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Request, error) {
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

	return s.store.Create(ctx, in)
}

func invalid(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, field)
}