package clickhouse

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Exec executes a query without returning rows
func Exec(query string, args ...interface{}) error {
	return CK.Exec(context.Background(), query, args...)
}

// Query executes a query that returns rows
func Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	return CK.Query(ctx, query, args...)
}

// QueryRow executes a query that returns a single row
func QueryRow(ctx context.Context, query string, args ...interface{}) driver.Row {
	return CK.QueryRow(ctx, query, args...)
}
