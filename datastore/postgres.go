package datastore

import (
	"database/sql"
	"fmt"

	sqlds "github.com/ipfs/go-ds-sql"
	"github.com/ipfs/go-ds-sql/postgres"
)

const tableName = "records"

// NewPostgreSQLDatastore creates a new sqlds.Datastore that talks to a PostgreSQL database
func NewPostgreSQLDatastore(connstr string) (*sqlds.Datastore, error) {
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key TEXT NOT NULL UNIQUE, data BYTEA NOT NULL)", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to init PostgreSQL database: %w", err)
	}

	return sqlds.NewDatastore(db, postgres.Queries{TableName: tableName}), nil
}
