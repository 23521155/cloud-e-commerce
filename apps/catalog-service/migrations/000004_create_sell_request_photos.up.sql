CREATE TABLE sell_request_photos (
    blob_name NVARCHAR(400) NOT NULL
        CONSTRAINT pk_sell_request_photos PRIMARY KEY,

    request_id BIGINT NOT NULL
        CONSTRAINT fk_sell_request_photos_request
            REFERENCES sell_requests (id) ON DELETE CASCADE,

    position TINYINT NOT NULL,
    content_type NVARCHAR(50) NOT NULL,
    size_bytes INT NOT NULL,

    width INT NULL,
    height INT NULL,
    processed_at DATETIME2 NULL,

    created_at DATETIME2 NOT NULL
        CONSTRAINT df_sell_request_photos_created_at
        DEFAULT SYSUTCDATETIME()
);

CREATE INDEX ix_sell_request_photos_request_id
    ON sell_request_photos (request_id, position);
