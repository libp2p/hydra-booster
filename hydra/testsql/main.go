package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
)

const TableName = "records"

func countProviderRecordsApproximately(ctx context.Context, pgxPool *pgxpool.Pool) error {
	var approxCountSql = `SELECT
	(reltuples/relpages) * (
	  pg_relation_size(%s) /
	  (current_setting('block_size')::integer)
	)
	FROM pg_class where relname = %s;`
	row := pgxPool.QueryRow(ctx, approxCountSql, TableName, TableName)
	var numProvRecords float64
	err := row.Scan(&numProvRecords)
	if err != nil {
		return err
	}
	fmt.Printf("found %d prov records\n", int64(numProvRecords))
	return nil
}

func main() {
	ctx := context.TODO()
	pool, err := pgxpool.Connect(ctx,
		"postgresql://doadmin:vy2dihngj33wc6o3@private-db-postgres-hydra-do-user-7378862-0.b.db.ondigitalocean.com:25060/defaultdb?sslmode=require")
	if err != nil {
		fmt.Printf("pg connect (%v)", err)
		os.Exit(1)
		return
	}
	err = countProviderRecordsApproximately(ctx, pool)
	if err != nil {
		fmt.Printf("query (%v)", err)
		os.Exit(1)
		return
	}
}
