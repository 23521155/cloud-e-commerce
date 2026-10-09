package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
)

// Open connects to SQL Server with a sqlserver:// URL and checks the
// connection before returning.
func Open(ctx context.Context, url string) (*sqlx.DB, error) {

	db, err := sqlx.Open("sqlserver", url)
	if err != nil {
		return nil, err
	}

	// Azure SQL serverless auto-pauses; the first connection after a
	// pause can take up to a minute while the database resumes.
	pingCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
