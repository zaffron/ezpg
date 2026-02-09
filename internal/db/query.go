package db

import (
	"context"
	"fmt"
	"time"
)

func (m *Manager) ExecQuery(ctx context.Context, connName, query string, nullDisplay string) (*QueryResult, error) {
	pool, err := m.Pool(connName)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	result := &QueryResult{
		Columns: make([]string, len(fields)),
	}
	for i, f := range fields {
		result.Columns[i] = f.Name
	}

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("reading row: %w", err)
		}

		row := make([]string, len(values))
		for i, v := range values {
			if v == nil {
				row[i] = nullDisplay
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	result.RowCount = len(result.Rows)
	result.ExecTime = time.Since(start)

	tag := rows.CommandTag()
	if !tag.Select() {
		result.Message = tag.String()
	}

	return result, nil
}

func (m *Manager) QueryTableData(ctx context.Context, connName, schema, table string, limit, offset int, nullDisplay string) (*QueryResult, error) {
	query := fmt.Sprintf(
		`SELECT * FROM %q.%q LIMIT %d OFFSET %d`,
		schema, table, limit, offset,
	)
	return m.ExecQuery(ctx, connName, query, nullDisplay)
}
