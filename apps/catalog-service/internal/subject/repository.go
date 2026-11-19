package subject

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// List returns every subject with its count of available books. The
// catch-all "uncategorized" subject is listed last.
func (r *Repository) List(ctx context.Context) ([]Subject, error) {

	const query = `
SELECT s.id, s.slug, s.name, COUNT(b.id) AS book_count
FROM subjects s
LEFT JOIN books b ON b.subject_id = s.id AND b.status = N'AVAILABLE'
GROUP BY s.id, s.slug, s.name
ORDER BY CASE WHEN s.slug = N'uncategorized' THEN 1 ELSE 0 END, s.name`

	subjects := []Subject{}
	if err := r.db.SelectContext(ctx, &subjects, query); err != nil {
		return nil, err
	}

	return subjects, nil
}
