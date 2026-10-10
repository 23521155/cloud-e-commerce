-- A book is a single copy, so a checkout has to hold it while the customer
-- pays. reserved_by is the holder (the order code) so one order can never
-- release or sell a copy that another order holds; reserved_until lets the
-- catalogue free a hold on its own if the order service never comes back.
-- A sold book keeps reserved_by: it records which order bought it, which
-- makes "sell" safe to repeat.

ALTER TABLE books ADD
    reserved_by    NVARCHAR(16) NULL,
    reserved_until DATETIME2    NULL;

-- golang-migrate sends this file as one batch and SQL Server compiles a
-- batch before running it, so statements that name the new columns would fail
-- with "Invalid column name". EXEC compiles them once the columns exist.

-- Nothing created holds before this migration, so any RESERVED row has no
-- holder and would break the constraint below.
EXEC(N'UPDATE books SET status = N''AVAILABLE'' WHERE status = N''RESERVED''');

EXEC(N'ALTER TABLE books ADD CONSTRAINT ck_books_reservation CHECK (
    (status = N''AVAILABLE'' AND reserved_by IS NULL     AND reserved_until IS NULL)
 OR (status = N''RESERVED''  AND reserved_by IS NOT NULL AND reserved_until IS NOT NULL)
 OR (status = N''SOLD''      AND reserved_until IS NULL)
)');

-- Finding the holds that have run out. Deliberately not a filtered index:
-- those make every UPDATE on books fail from a session with
-- QUOTED_IDENTIFIER off (sqlcmd and several admin tools default to that).
EXEC(N'CREATE INDEX ix_books_status_reserved_until ON books (status, reserved_until)');
