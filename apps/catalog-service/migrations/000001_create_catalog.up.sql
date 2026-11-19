-- Text columns are NVARCHAR because go-mssqldb sends Go strings as
-- nvarchar; comparing them with VARCHAR columns would force an implicit
-- conversion and skip the index.

CREATE TABLE subjects (
    id   INT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_subjects PRIMARY KEY,
    slug NVARCHAR(100)      NOT NULL CONSTRAINT uq_subjects_slug UNIQUE,
    name NVARCHAR(100)      NOT NULL
);

CREATE TABLE books (
    id           INT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_books PRIMARY KEY,
    asin         NVARCHAR(10)       NOT NULL CONSTRAINT uq_books_asin UNIQUE,
    slug         NVARCHAR(100)      NOT NULL CONSTRAINT uq_books_slug UNIQUE,
    title        NVARCHAR(600)      NOT NULL,
    author       NVARCHAR(150)      NULL,
    year         SMALLINT           NOT NULL,
    publisher    NVARCHAR(200)      NULL,
    edition      NVARCHAR(200)      NULL,
    binding      NVARCHAR(50)       NULL,
    language     NVARCHAR(50)       NULL,
    pages        INT                NULL,
    price_vnd    BIGINT             NOT NULL CONSTRAINT ck_books_price_vnd CHECK (price_vnd > 0),
    rating       DECIMAL(2, 1)      NULL,
    rating_count INT                NOT NULL CONSTRAINT df_books_rating_count DEFAULT 0,
    description  NVARCHAR(MAX)      NULL,
    cover_url    NVARCHAR(300)      NULL,
    subject_id   INT                NOT NULL CONSTRAINT fk_books_subject REFERENCES subjects (id),
    rare         BIT                NOT NULL CONSTRAINT df_books_rare DEFAULT 0,
    status       NVARCHAR(10)       NOT NULL CONSTRAINT df_books_status DEFAULT N'AVAILABLE'
                                    CONSTRAINT ck_books_status CHECK (status IN (N'AVAILABLE', N'RESERVED', N'SOLD')),
    -- Lowercased title + author without diacritics, for LIKE search.
    search_text  NVARCHAR(800)      NOT NULL,
    created_at   DATETIME2          NOT NULL CONSTRAINT df_books_created_at DEFAULT SYSUTCDATETIME(),
    updated_at   DATETIME2          NOT NULL CONSTRAINT df_books_updated_at DEFAULT SYSUTCDATETIME()
);

CREATE INDEX ix_books_subject_id ON books (subject_id);
CREATE INDEX ix_books_price_vnd ON books (price_vnd);
CREATE INDEX ix_books_year ON books (year);
