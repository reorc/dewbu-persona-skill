package db

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Column represents a db9 column descriptor.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// QueryResult represents the raw JSON output from db9 db sql --json.
type QueryResult struct {
	Columns  []Column        `json:"columns"`
	Rows     [][]interface{} `json:"rows"`
	RowCount int             `json:"row_count"`
	Command  string          `json:"command"`
}

// ColumnNames returns just the column name strings.
func (r *QueryResult) ColumnNames() []string {
	names := make([]string, len(r.Columns))
	for i, c := range r.Columns {
		names[i] = c.Name
	}
	return names
}

// Query executes a SQL query against db9 and returns raw result.
func Query(database, sql string) (*QueryResult, error) {
	cmd := exec.Command("db9", "db", "sql", database, "-q", sql, "--json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("db9 error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("db9 exec error: %w", err)
	}

	var result QueryResult
	if err := json.Unmarshal(out, &result); err != nil {
		// db9 might return non-JSON (plain text table format)
		// Try to parse as plain text
		return nil, fmt.Errorf("failed to parse db9 JSON output: %w\nraw: %s", err, string(out[:min(len(out), 200)]))
	}
	return &result, nil
}

// QueryRows executes SQL and returns results as []map[string]interface{}.
func QueryRows(database, sql string) ([]map[string]interface{}, error) {
	result, err := Query(database, sql)
	if err != nil {
		return nil, err
	}

	colNames := result.ColumnNames()
	rows := make([]map[string]interface{}, 0, len(result.Rows))
	for _, row := range result.Rows {
		m := make(map[string]interface{}, len(colNames))
		for i, col := range colNames {
			if i < len(row) {
				m[col] = row[i]
			}
		}
		rows = append(rows, m)
	}
	return rows, nil
}

// Exec executes a SQL statement (no result expected).
func Exec(database, sql string) error {
	cmd := exec.Command("db9", "db", "sql", database, "-q", sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("db9 error: %s", string(out))
	}
	return nil
}

// QueryScalar executes SQL and returns a single value.
func QueryScalar(database, sql string) (interface{}, error) {
	result, err := Query(database, sql)
	if err != nil {
		return nil, err
	}
	if len(result.Rows) == 0 || len(result.Rows[0]) == 0 {
		return nil, nil
	}
	return result.Rows[0][0], nil
}

// EscapeString escapes a string for safe SQL interpolation.
func EscapeString(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "\x00", "")
	return "'" + s + "'"
}

// EscapeArray formats a Go string slice as a Postgres ARRAY literal.
func EscapeArray(items []string) string {
	escaped := make([]string, len(items))
	for i, item := range items {
		escaped[i] = EscapeString(item)
	}
	return "ARRAY[" + strings.Join(escaped, ",") + "]"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
