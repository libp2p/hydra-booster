package datastore

import (
	"github.com/ipfs/go-datastore"
	"github.com/jackc/pgx/v4/pgxpool"
)

type WithPgxPool interface {
	PgxPool() *pgxpool.Pool
}

type BatchingWithPgxPool struct {
	Pool WithPgxPool
	datastore.Batching
}

func (x BatchingWithPgxPool) PgxPool() *pgxpool.Pool {
	return x.Pool.PgxPool()
}
