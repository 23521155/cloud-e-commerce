DROP INDEX IF EXISTS ix_books_status_reserved_until ON books;
ALTER TABLE books DROP CONSTRAINT IF EXISTS ck_books_reservation;
ALTER TABLE books DROP COLUMN IF EXISTS reserved_until;
ALTER TABLE books DROP COLUMN IF EXISTS reserved_by;
