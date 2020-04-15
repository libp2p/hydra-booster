package datastore

import (
	"database/sql"
	"fmt"

	sqlds "github.com/alanshaw/sql-datastore"
	"github.com/alanshaw/sql-datastore/postgres"
)

// NewPostgreSQLDatastore creates a new sqlds.Datastore that talks to a PostgreSQL database
func NewPostgreSQLDatastore(connstr string) (*sqlds.Datastore, error) {
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS blocks (key TEXT NOT NULL UNIQUE, data BYTEA NOT NULL)")
	if err != nil {
		return nil, fmt.Errorf("failed to init PostgreSQL database: %w", err)
	}

	return sqlds.NewDatastore(db, postgres.Queries{}), nil
}
