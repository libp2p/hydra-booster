package datastore

import (
	"context"
	"fmt"
	"os"

	pgds "github.com/ipfs/ipfs-ds-postgres"
	"github.com/jackc/pgx/v4/pgxpool"
)

const TableName = "records"

// NewPostgreSQLDatastore creates a new pgds.Datastore that talks to a PostgreSQL database
func NewPostgreSQLDatastore(ctx context.Context, connstr string, createDB bool) (*pgds.Datastore, error) {
	if createDB {
		connConf, err := pgxpool.ParseConfig(connstr)
		if err != nil {
			return nil, err
		}
		pool, err := pgxpool.ConnectConfig(ctx, connConf)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "Creating Table\n")
		_, err = pool.Exec(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key TEXT NOT NULL UNIQUE, data BYTEA)", TableName))
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "Creating Index\n")
		_, err = pool.Exec(ctx, fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_key_text_pattern_ops_idx ON %s (key text_pattern_ops)", TableName, TableName))
		if err != nil {
			return nil, err
		}
		pool.Close()
	}
	fmt.Fprintf(os.Stderr, "Connecting to Database\n")
	ds, err := pgds.NewDatastore(ctx, connstr, pgds.Table(TableName))
	if err != nil {
		return nil, err
	}
	return ds, nil
}
