package sell

import(
	"context"
	"database/sql"

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

// nullableText stores empty optional fields as SQL NULL.
func nullableText(value string) sql.NullString {
	return sql.NullString{
		String: value,
		Valid:  value != "",
	}
}