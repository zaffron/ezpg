package db

import (
	"context"
	"fmt"
)

func (m *Manager) ListTables(ctx context.Context, connName string) ([]TableInfo, error) {
	pool, err := m.Pool(connName)
	if err != nil {
		return nil, err
	}

	/**
	* So, this function basically does the following:
	* 1. after geting the connection pool, it executes a query to get the table information
	* 2. it filters the results by excluding system tables and only returning "real" tables
	* 3. it orders the results by the schema and table name
	* 4. it returns a slice of TableInfo structs containing the table information
	 */
	rows, err := pool.Query(ctx, `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
			AND table_type = 'BASE TABLE'
		ORDER BY table_schema, table_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Schema, &t.Name); err != nil {
			return nil, fmt.Errorf("scanning table: %w", err)
		}
		tables = append(tables, t)
	}

	return tables, rows.Err()
}

func (m *Manager) ListColumns(ctx context.Context, connName string, schema, table string) ([]ColumnInfo, error) {
	pool, err := m.Pool(connName)
	if err != nil {
		return nil, err
	}

	/**
	* So, this function basically does the following:
	* 1. after geting the connection pool, it executes a query to get the column information
	* 2. it filters the results by the schema and table name
	* 3. it orders the results by the column position
	* 4. it uses a subquery to check if the column is part of a primary key
	* 5. it returns a slice of ColumnInfo structs containing the column information
	* Good fucking lord, I hope I don't have to write queries like this XD, pgAdmin basically was keeping me dumb
	 */
	rows, err := pool.Query(ctx, `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES',
			COALESCE(
				(
					SELECT true FROM information_schema.key_column_usage kcu
						JOIN information_schema.table_constraints tc
							ON kcu.constrain_name = tc.constrain_name
							AND kcu.table_shcema = tc.table_schema
						WHERE tc.constraint_type = 'PRIMARY KEY'
							AND kcu.table_schema = c.table_schema
							AND kcu.table_name = c.table_name
							AND kcu.column_name = c.column_name
					LIMIT 1
				),
				false
			) as is_primary
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`, schema, table)

	if err != nil {
		return nil, fmt.Errorf("listing columns: %w", err)
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		if err := rows.Scan(&c.Name, &c.DataType, &c.IsNullable, &c.IsPrimary); err != nil {
			return nil, fmt.Errorf("scanning column: %w", err)
		}
		cols = append(cols, c)
	}

	return cols, rows.Err()

}
