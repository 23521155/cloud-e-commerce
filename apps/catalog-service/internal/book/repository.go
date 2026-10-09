package book

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("book not found")

const PageSize = 8

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

const selectBook = `
SELECT b.id, b.asin, b.slug, b.title, b.author, b.year, b.publisher, b.edition,
       b.binding, b.language, b.pages, b.price_vnd, b.rating, b.rating_count,
       b.description, b.cover_url, b.rare, b.status,
       s.slug AS subject_slug, s.name AS subject_name
FROM books b
JOIN subjects s ON s.id = b.subject_id`

func (r *Repository) GetBySlug(ctx context.Context, slug string) (Book, error) {
	var b Book
	err := r.db.GetContext(ctx, &b, selectBook+" WHERE b.slug = @slug", sql.Named("slug", slug))
	if err != nil {
		if err == sql.ErrNoRows {
			return Book{}, ErrNotFound
		}
		return Book{}, err
	}
	return b, err
}

func buildWhere(f Filter) (string, []any) {
	where := []string{"b.status = N'AVAILABLE'"}
	var args []any

	if f.Subject != "" {
		where = append(where, "s.slug = @subject")
		args = append(args, sql.Named("subject", f.Subject))
	}
	if f.Rare {
		where = append(where, "b.rare = 1")
	}
	for i, term := range f.Terms {
		name := fmt.Sprintf("q%d", i)
		where = append(where, "b.search_text LIKE @"+name+` ESCAPE '\'`)
		args = append(args, sql.Named(name, "%"+escapeLike(term)+"%"))
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

// likeEscaper escapes the LIKE wildcards so user input matches literally.
// The backslash comes first so the escapes added after it are not doubled.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`, `[`, `\[`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

// orderBy maps the public sort keys to SQL. It is a whitelist: the sort
// value comes from the client and must never be concatenated into SQL.
// Every sort ends with b.id so paging is stable when values tie.
var orderBy = map[string]string{
	"new":        "b.id DESC",
	"price-asc":  "b.price_vnd ASC, b.id DESC",
	"price-desc": "b.price_vnd DESC, b.id DESC",
	"year":       "b.year ASC, b.id DESC",
}

// List returns one page of available books and the total number of matches.
func (r *Repository) List(ctx context.Context, f Filter) ([]Book, int, error) {

	where, args := buildWhere(f)

	// Count with the filter args only, before offset/limit are added.
	var total int
	countQuery := `SELECT COUNT(*) FROM books b JOIN subjects s ON s.id = b.subject_id` + where
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	order, ok := orderBy[f.Sort]
	if !ok {
		order = orderBy["new"]
	}

	page := max(f.Page, 1)

	query := selectBook + where +
		" ORDER BY " + order +
		" OFFSET @offset ROWS FETCH NEXT @limit ROWS ONLY"

	args = append(args,
		sql.Named("offset", (page-1)*PageSize),
		sql.Named("limit", PageSize),
	)

	books := []Book{}
	if err := r.db.SelectContext(ctx, &books, query, args...); err != nil {
		return nil, 0, err
	}

	return books, total, nil
}

// Related returns up to limit other available books, those sharing the
// subject of slug first. The caller checks that slug exists.
func (r *Repository) Related(ctx context.Context, slug string, limit int) ([]Book, error) {

	query := strings.Replace(selectBook, "SELECT ", "SELECT TOP (@limit) ", 1) + `
WHERE b.status = N'AVAILABLE' AND b.slug <> @slug
ORDER BY CASE WHEN b.subject_id = (SELECT subject_id FROM books WHERE slug = @slug) THEN 0 ELSE 1 END,
         b.id DESC`

	books := []Book{}
	err := r.db.SelectContext(ctx, &books, query,
		sql.Named("slug", slug),
		sql.Named("limit", limit),
	)
	if err != nil {
		return nil, err
	}

	return books, nil
}

// BySlugs returns the books with the given slugs whatever their status, so a
// basket can still show a book that has since been sold. The order is not
// defined and unknown slugs are left out.
func (r *Repository) BySlugs(ctx context.Context, slugs []string) ([]Book, error) {

	if len(slugs) == 0 {
		return []Book{}, nil
	}

	placeholders := make([]string, len(slugs))
	args := make([]any, len(slugs))
	for i, slug := range slugs {
		name := fmt.Sprintf("s%d", i)
		placeholders[i] = "@" + name
		args[i] = sql.Named(name, slug)
	}

	query := selectBook + " WHERE b.slug IN (" + strings.Join(placeholders, ", ") + ")"

	books := []Book{}
	if err := r.db.SelectContext(ctx, &books, query, args...); err != nil {
		return nil, err
	}

	return books, nil
}
