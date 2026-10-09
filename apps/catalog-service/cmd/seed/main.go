// Command seed upserts seed/books.jsonl.gz into the catalog database.
// It is safe to run repeatedly: rows are matched on subjects.slug and
// books.asin. Run migrations first.
//
//	go run ./cmd/seed
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/fizzisme/catalog-service/internal/config"
	"github.com/fizzisme/catalog-service/internal/database"
	"github.com/fizzisme/catalog-service/internal/seeddata"
	"github.com/jmoiron/sqlx"
)

// bookColumns are the columns written by the seed, in parameter order.
// status is left out so re-seeding never resets a sold book.
var bookColumns = []string{
	"asin", "slug", "title", "author", "year", "publisher", "edition",
	"binding", "language", "pages", "price_vnd", "rating", "rating_count",
	"description", "cover_url", "subject_id", "rare", "search_text",
}

// SQL Server allows at most 2100 parameters per statement.
const batchSize = 2000 / 18

func main() {

	file := flag.String("file", "seed/books.jsonl.gz", "cleaned seed file from prepare-seed")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	books, err := seeddata.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// One transaction, so a failed run leaves the database unchanged.
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	subjectIDs, err := upsertSubjects(ctx, tx, books)
	if err != nil {
		log.Fatal(err)
	}

	if err := upsertBooks(ctx, tx, books, subjectIDs); err != nil {
		log.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	log.Printf("seeded %d subjects and %d books", len(subjectIDs), len(books))
}

// upsertSubjects writes every distinct subject and returns their IDs by slug.
func upsertSubjects(ctx context.Context, tx *sqlx.Tx, books []seeddata.Book) (map[string]int, error) {

	const query = `
MERGE subjects AS t
USING (VALUES (@p1, @p2)) AS s (slug, name)
ON t.slug = s.slug
WHEN MATCHED THEN UPDATE SET name = s.name
WHEN NOT MATCHED THEN INSERT (slug, name) VALUES (s.slug, s.name);`

	seen := map[string]bool{}
	for _, b := range books {
		if seen[b.SubjectSlug] {
			continue
		}
		seen[b.SubjectSlug] = true

		if _, err := tx.ExecContext(ctx, query, b.SubjectSlug, b.SubjectName); err != nil {
			return nil, fmt.Errorf("subject %q: %w", b.SubjectSlug, err)
		}
	}

	var rows []struct {
		ID   int    `db:"id"`
		Slug string `db:"slug"`
	}
	if err := tx.SelectContext(ctx, &rows, "SELECT id, slug FROM subjects"); err != nil {
		return nil, err
	}

	ids := make(map[string]int, len(rows))
	for _, r := range rows {
		ids[r.Slug] = r.ID
	}
	return ids, nil
}

// upsertBooks merges books in batches keyed on asin.
func upsertBooks(ctx context.Context, tx *sqlx.Tx, books []seeddata.Book, subjectIDs map[string]int) error {

	for start := 0; start < len(books); start += batchSize {

		batch := books[start:min(start+batchSize, len(books))]

		args := make([]any, 0, len(batch)*len(bookColumns))
		for _, b := range batch {
			args = append(args,
				b.ASIN, b.Slug, b.Title, b.Author, b.Year, b.Publisher, b.Edition,
				b.Binding, b.Language, b.Pages, b.PriceVND, b.Rating, b.RatingCount,
				b.Description, b.CoverURL, subjectIDs[b.SubjectSlug], b.Rare,
				seeddata.SearchText(b),
			)
		}

		if _, err := tx.ExecContext(ctx, mergeBooksQuery(len(batch)), args...); err != nil {
			return fmt.Errorf("books %d-%d: %w", start, start+len(batch)-1, err)
		}
	}
	return nil
}

// mergeBooksQuery builds a MERGE with one VALUES row per book.
func mergeBooksQuery(rows int) string {

	var values strings.Builder
	p := 1
	for i := range rows {
		if i > 0 {
			values.WriteString(",\n")
		}
		values.WriteString("(")
		for j := range bookColumns {
			if j > 0 {
				values.WriteString(", ")
			}
			fmt.Fprintf(&values, "@p%d", p)
			p++
		}
		values.WriteString(")")
	}

	sets := make([]string, 0, len(bookColumns))
	sources := make([]string, 0, len(bookColumns))
	for _, c := range bookColumns {
		sources = append(sources, "s."+c)
		if c != "asin" {
			sets = append(sets, c+" = s."+c)
		}
	}

	columns := strings.Join(bookColumns, ", ")

	return fmt.Sprintf(`
MERGE books AS t
USING (VALUES %s) AS s (%s)
ON t.asin = s.asin
WHEN MATCHED THEN UPDATE SET %s, updated_at = SYSUTCDATETIME()
WHEN NOT MATCHED THEN INSERT (%s) VALUES (%s);`,
		values.String(), columns, strings.Join(sets, ", "), columns, strings.Join(sources, ", "))
}
