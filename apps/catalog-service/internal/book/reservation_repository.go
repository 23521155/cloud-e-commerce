package book

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	mssql "github.com/microsoft/go-mssqldb"
)

// errShort aborts a reservation transaction when some book could not be held.
var errShort = errors.New("not every book could be held")

// sqlDeadlock is the SQL Server error number of a deadlock victim.
const sqlDeadlock = 1205

// idList returns the "@id0, @id1, ..." placeholders and their arguments.
func idList(ids []int) (string, []any) {

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		name := fmt.Sprintf("id%d", i)
		placeholders[i] = "@" + name
		args[i] = sql.Named(name, id)
	}

	return strings.Join(placeholders, ", "), args
}

// slugList returns the "@slug0, @slug1, ..." placeholders and their arguments.
func slugList(slugs []string) (string, []any) {

	placeholders := make([]string, len(slugs))
	args := make([]any, len(slugs))
	for i, slug := range slugs {
		name := fmt.Sprintf("slug%d", i)
		placeholders[i] = "@" + name
		args[i] = sql.Named(name, slug)
	}

	return strings.Join(placeholders, ", "), args
}

// inTx runs fn in a transaction, committing when it returns nil. Two
// orders locking overlapping books can deadlock; SQL Server then rolls one
// back and it is safe to simply run it again.
func (r *Repository) inTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {

	var err error
	for attempt := 0; attempt < 2; attempt++ {

		err = r.runTx(ctx, fn)

		var sqlErr mssql.Error
		if !errors.As(err, &sqlErr) || sqlErr.Number != sqlDeadlock {
			return err
		}
	}

	return err
}

func (r *Repository) runTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Reserve holds every book in slugs for holder for ttlSeconds, or none of them.
// The decision is one UPDATE, so two orders racing for the same copy cannot
// both win: the second sees RESERVED and gets an UnavailableError. A hold
// that has run out, or one the same holder already has, can be taken again.
// slugs must be unique and sorted so concurrent orders lock in the same order.
func (r *Repository) Reserve(ctx context.Context, holder string, ttlSeconds int, slugs []string) ([]Held, error) {

	list, args := slugList(slugs)
	args = append(args,
		sql.Named("holder", holder),
		sql.Named("ttl", int32(ttlSeconds)),
	)

	query := `
UPDATE books
SET status = N'RESERVED',
    reserved_by = @holder,
    reserved_until = DATEADD(SECOND, @ttl, SYSUTCDATETIME()),
    updated_at = SYSUTCDATETIME()
OUTPUT inserted.id, inserted.slug, inserted.title, inserted.author,
       inserted.cover_url, inserted.price_vnd, inserted.reserved_until
WHERE slug IN (` + list + `)
  AND (status = N'AVAILABLE'
       OR (status = N'RESERVED' AND (reserved_by = @holder OR reserved_until < SYSUTCDATETIME())))`

	var held []Held
	err := r.inTx(ctx, func(tx *sqlx.Tx) error {

		held = []Held{}
		if err := tx.SelectContext(ctx, &held, query, args...); err != nil {
			return err
		}

		if len(held) != len(slugs) {
			return errShort
		}

		return nil
	})

	if errors.Is(err, errShort) {
		return nil, r.unavailableBySlug(ctx, slugs, func(row bookState) bool {
			for _, h := range held {
				if h.ID == row.ID {
					return true
				}
			}
			return false
		})
	}
	if err != nil {
		return nil, err
	}

	return held, nil
}

// Release gives back the books holder is holding. Books held by someone
// else, sold, or already free are left alone, so it is safe to call twice.
func (r *Repository) Release(ctx context.Context, holder string, ids []int) (int, error) {

	list, args := idList(ids)
	args = append(args, sql.Named("holder", holder))

	res, err := r.db.ExecContext(ctx, `
UPDATE books
SET status = N'AVAILABLE', reserved_by = NULL, reserved_until = NULL,
    updated_at = SYSUTCDATETIME()
WHERE id IN (`+list+`) AND status = N'RESERVED' AND reserved_by = @holder`, args...)
	if err != nil {
		return 0, err
	}

	n, err := res.RowsAffected()
	return int(n), err
}

// Sell turns the books holder holds into SOLD, all or none. A book this
// holder has already sold counts as done, so a repeated payment
// notification does not fail. ids must be unique and sorted.
func (r *Repository) Sell(ctx context.Context, holder string, ids []int) error {

	list, args := idList(ids)
	args = append(args, sql.Named("holder", holder))

	soldToHolder := func(row bookState) bool {
		return row.Status == StatusSold && row.ReservedBy != nil && *row.ReservedBy == holder
	}

	err := r.inTx(ctx, func(tx *sqlx.Tx) error {

		if _, err := tx.ExecContext(ctx, `
UPDATE books
SET status = N'SOLD', reserved_until = NULL, updated_at = SYSUTCDATETIME()
WHERE id IN (`+list+`) AND status = N'RESERVED' AND reserved_by = @holder`, args...); err != nil {
			return err
		}

		var rows []bookState
		if err := tx.SelectContext(ctx, &rows,
			"SELECT id, status, reserved_by FROM books WHERE id IN ("+list+")", args[:len(ids)]...); err != nil {
			return err
		}

		if len(unavailableOf(ids, rows, soldToHolder)) > 0 {
			return errShort
		}

		return nil
	})

	if errors.Is(err, errShort) {
		return r.unavailableByID(ctx, ids, soldToHolder)
	}

	return err
}

// ReleaseExpired frees every hold whose time has run out and returns how
// many books were freed.
func (r *Repository) ReleaseExpired(ctx context.Context) (int, error) {

	res, err := r.db.ExecContext(ctx, `
UPDATE books
SET status = N'AVAILABLE', reserved_by = NULL, reserved_until = NULL,
    updated_at = SYSUTCDATETIME()
WHERE status = N'RESERVED' AND reserved_until < SYSUTCDATETIME()`)
	if err != nil {
		return 0, err
	}

	n, err := res.RowsAffected()
	return int(n), err
}

// bookState is the part of a book a failed reservation needs to explain itself.
type bookState struct {
	ID         int     `db:"id"`
	Slug       string  `db:"slug"`
	Status     string  `db:"status"`
	ReservedBy *string `db:"reserved_by"`
}

// unavailableByID reads the current state of ids, after the failed
// transaction has been rolled back, and reports the ones that are not ok.
func (r *Repository) unavailableByID(ctx context.Context, ids []int, ok func(bookState) bool) error {

	list, args := idList(ids)

	var rows []bookState
	if err := r.db.SelectContext(ctx, &rows,
		"SELECT id, slug, status, reserved_by FROM books WHERE id IN ("+list+")", args...); err != nil {
		return err
	}

	return &UnavailableError{Books: unavailableOf(ids, rows, ok)}
}

// unavailableBySlug is unavailableByID for a request made with slugs.
func (r *Repository) unavailableBySlug(ctx context.Context, slugs []string, ok func(bookState) bool) error {

	list, args := slugList(slugs)

	var rows []bookState
	if err := r.db.SelectContext(ctx, &rows,
		"SELECT id, slug, status, reserved_by FROM books WHERE slug IN ("+list+")", args...); err != nil {
		return err
	}

	return &UnavailableError{Books: unavailableOfSlugs(slugs, rows, ok)}
}

// unavailableOf lists, in the order of ids, the books ok rejects. An id with
// no row is reported as MISSING, with no slug: it is not known.
func unavailableOf(ids []int, rows []bookState, ok func(bookState) bool) []Unavailable {

	byID := make(map[int]bookState, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	out := []Unavailable{}
	for _, id := range ids {
		row, found := byID[id]
		switch {
		case !found:
			out = append(out, Unavailable{ID: id, Status: StatusMissing})
		case !ok(row):
			out = append(out, Unavailable{ID: id, Slug: row.Slug, Status: row.Status})
		}
	}

	return out
}

// unavailableOfSlugs lists, in the order of slugs, the books ok rejects. A
// slug with no row is reported as MISSING, with no id: it is not known.
//
// SQL Server compares slugs without regard to case, so rows are matched the
// same way; the slug reported is the one the caller asked for.
func unavailableOfSlugs(slugs []string, rows []bookState, ok func(bookState) bool) []Unavailable {

	bySlug := make(map[string]bookState, len(rows))
	for _, row := range rows {
		bySlug[strings.ToLower(row.Slug)] = row
	}

	out := []Unavailable{}
	for _, slug := range slugs {
		row, found := bySlug[strings.ToLower(slug)]
		switch {
		case !found:
			out = append(out, Unavailable{Slug: slug, Status: StatusMissing})
		case !ok(row):
			out = append(out, Unavailable{ID: row.ID, Slug: slug, Status: row.Status})
		}
	}

	return out
}
