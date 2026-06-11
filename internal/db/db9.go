package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
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

type Config struct {
	Backend string
	APIURL  string
	APIKey  string
	Timeout time.Duration
}

var config = Config{
	Backend: getenv("DEWBU_BACKEND", "db9"),
	APIURL:  os.Getenv("DEWBU_API_BASE_URL"),
	APIKey:  os.Getenv("DEWBU_API_KEY"),
	Timeout: 30 * time.Second,
}

func Configure(cfg Config) {
	if cfg.Backend != "" {
		config.Backend = cfg.Backend
	}
	if cfg.APIURL != "" {
		config.APIURL = cfg.APIURL
	}
	if cfg.APIKey != "" {
		config.APIKey = cfg.APIKey
	}
	if cfg.Timeout > 0 {
		config.Timeout = cfg.Timeout
	}
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
	switch strings.ToLower(strings.TrimSpace(config.Backend)) {
	case "", "db9":
		return queryDB9(database, sql)
	case "http", "api":
		return queryHTTP(database, sql)
	default:
		return nil, fmt.Errorf("unknown backend %q; use db9 or http", config.Backend)
	}
}

func queryDB9(database, sql string) (*QueryResult, error) {
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

func queryHTTP(database, sql string) (*QueryResult, error) {
	if config.APIURL == "" {
		return nil, fmt.Errorf("DEWBU_API_BASE_URL is required when DEWBU_BACKEND=http")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("DEWBU_API_KEY is required when DEWBU_BACKEND=http")
	}
	endpoint := strings.TrimRight(config.APIURL, "/") + "/v1/query"
	body, err := json.Marshal(map[string]string{
		"database": database,
		"sql":      sql,
	})
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: config.Timeout}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http backend error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody struct {
			Error string `json:"error"`
		}
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errBody); decodeErr == nil && errBody.Error != "" {
			return nil, fmt.Errorf("http backend error: %s", errBody.Error)
		}
		return nil, fmt.Errorf("http backend returned status %d", resp.StatusCode)
	}
	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse http backend JSON output: %w", err)
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
	if strings.EqualFold(strings.TrimSpace(config.Backend), "http") || strings.EqualFold(strings.TrimSpace(config.Backend), "api") {
		return fmt.Errorf("Exec is not supported by the http backend")
	}
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

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
