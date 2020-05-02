package datastore

import (
	"database/sql"
	"fmt"
	"time"

	sqlds "github.com/ipfs/go-ds-sql"
	"github.com/ipfs/go-ds-sql/postgres"
	_ "github.com/jackc/pgx/v4/stdlib" // postgres driver
)

const (
	tableName       = "records"
	maxIdleConns    = 25 // TODO tie to number of heads?
	connMaxLifetime = time.Hour
)

// NewPostgreSQLDatastore creates a new sqlds.Datastore that talks to a PostgreSQL database
func NewPostgreSQLDatastore(connstr string) (*sqlds.Datastore, error) {
	db, err := sql.Open("pgx", connstr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key TEXT NOT NULL UNIQUE, data BYTEA NOT NULL)", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to init PostgreSQL database: %w", err)
	}

	return sqlds.NewDatastore(db, postgres.Queries{TableName: tableName}), nil
}
