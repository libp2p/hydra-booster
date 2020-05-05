package datastore

import (
	"context"
	"fmt"

	pgds "github.com/alanshaw/ipfs-ds-postgres"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const tableName = "records"

// NewPostgreSQLDatastore creates a new sqlds.Datastore that talks to a PostgreSQL database
func NewPostgreSQLDatastore(ctx context.Context, connstr string) (*pgds.Datastore, error) {
	connConf, err := pgx.ParseConfig(connstr)
	if err != nil {
		return nil, err
	}
	conn, err := pgx.ConnectConfig(context.Background(), connConf)
	if err != nil {
		return nil, err
	}
	_, err = conn.Exec(context.Background(), fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (key TEXT NOT NULL UNIQUE, data BYTEA NOT NULL)", tableName))
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.Connect(ctx, connstr)
	if err != nil {
		return nil, err
	}
	ds, err := pgds.NewDatastore(connstr, pgds.Table(tableName), pgds.Pool(pool))
	if err != nil {
		return nil, err
	}
	return ds, nil
}
