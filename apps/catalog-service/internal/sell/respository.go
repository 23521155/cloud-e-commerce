package sell

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create stores a submission with a server-controlled review status.
func (r *Repository) Create(ctx context.Context, in CreateInput) (Request, error) {
	const query = `
		INSERT INTO sell_requests (
			title, author, year, edition, binding, grade,
			faults, virtues, asking_price_vnd,
			seller_name, seller_email, seller_phone, status
		)
		OUTPUT
			INSERTED.id,
			INSERTED.title,
			INSERTED.author,
			INSERTED.year,
			INSERTED.edition,
			INSERTED.binding,
			INSERTED.grade,
			INSERTED.faults,
			INSERTED.virtues,
			INSERTED.asking_price_vnd,
			INSERTED.seller_name,
			INSERTED.seller_email,
			INSERTED.seller_phone,
			INSERTED.status,
			INSERTED.created_at,
			INSERTED.updated_at
		VALUES (
			@title, @author, @year, @edition, @binding, @grade,
			@faults, @virtues, @asking,
			@name, @email, @phone, @status
		)`
	
	var request Request
	err := r.db.GetContext(ctx, &request, query,
		sql.Named("title", in.Title),
		sql.Named("author", in.Author),
		sql.Named("year", in.Year),
		sql.Named("edition", nullableText(in.Edition)),
		sql.Named("binding", in.Binding),
		sql.Named("grade", in.Grade),
		sql.Named("faults", in.Faults),
		sql.Named("virtues", nullableText(in.Virtues)),
		sql.Named("asking", in.Asking),
		sql.Named("name", in.Name),
		sql.Named("email", in.Email),
		sql.Named("phone", nullableText(in.Phone)),
		sql.Named("status", StatusPendingReview),
	)
	if err != nil {
		return Request{}, err
	}

	return request, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sell_requests WHERE id = @id`, sql.Named("id", id))
	return err
}

func (r *Repository) List(ctx context.Context, limit int) ([]Request, error) {
	var requests []Request
	err := r.db.SelectContext(ctx, &requests, `
		SELECT TOP (@limit)
			id, title, author, year, edition, binding, grade,
			faults, virtues, asking_price_vnd,
			seller_name, seller_email, seller_phone, status,
			created_at, updated_at
		FROM sell_requests
		ORDER BY created_at DESC, id DESC`,
		sql.Named("limit", limit),
	)
	if err != nil || len(requests) == 0 {
		return requests, err
	}

	var photos []Photo
	err = r.db.SelectContext(ctx, &photos, `
		SELECT p.blob_name, p.request_id, p.position, p.content_type, p.size_bytes,
			p.width, p.height, p.processed_at
		FROM sell_request_photos p
		WHERE p.request_id >= @oldest
		ORDER BY p.request_id, p.position`,
		sql.Named("oldest", requests[len(requests)-1].ID),
	)
	if err != nil {
		return nil, err
	}

	byRequest := make(map[int64][]Photo, len(requests))
	for _, p := range photos {
		byRequest[p.RequestID] = append(byRequest[p.RequestID], p)
	}
	for i := range requests {
		requests[i].Photos = byRequest[requests[i].ID]
		if requests[i].Photos == nil {
			requests[i].Photos = []Photo{}
		}
	}
	return requests, nil
}

func (r *Repository) Photo(ctx context.Context, requestID int64, position int) (Photo, error) {
	var photo Photo
	err := r.db.GetContext(ctx, &photo, `
		SELECT blob_name, request_id, position, content_type, size_bytes, width, height, processed_at
		FROM sell_request_photos
		WHERE request_id = @request AND position = @position`,
		sql.Named("request", requestID),
		sql.Named("position", position),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Photo{}, ErrPhotoNotFound
	}
	return photo, err
}

// nullableText stores empty optional fields as SQL NULL.
func nullableText(value string) sql.NullString {
	return sql.NullString{
		String: value,
		Valid:  value != "",
	}
}