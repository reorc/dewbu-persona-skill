package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	Backend: getenv("DEWBU_BACKEND", "http"),
	APIURL:  os.Getenv("DEWBU_API_BASE_URL"),
	APIKey:  os.Getenv("DEWBU_API_KEY"),
	Timeout: 30 * time.Second,
}

type fileConfig struct {
	Backend        string `json:"backend,omitempty"`
	SvcBaseURL     string `json:"svc_base_url,omitempty"`
	APIBaseURL     string `json:"api_base_url,omitempty"`
	APIKey         string `json:"api_key,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
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

func DefaultConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("DEWBU_CONFIG")); path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".dewbu", "config.json")
	}
	return filepath.Join(home, ".dewbu", "config.json")
}

func LoadConfigFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var raw fileConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	apiURL := strings.TrimSpace(raw.SvcBaseURL)
	if apiURL == "" {
		apiURL = strings.TrimSpace(raw.APIBaseURL)
	}
	cfg := Config{
		Backend: strings.TrimSpace(raw.Backend),
		APIURL:  apiURL,
		APIKey:  strings.TrimSpace(raw.APIKey),
	}
	if raw.TimeoutSeconds > 0 {
		cfg.Timeout = time.Duration(raw.TimeoutSeconds) * time.Second
	}
	if cfg.Backend == "" && cfg.APIURL != "" && cfg.APIKey != "" {
		cfg.Backend = "http"
	}
	return cfg, nil
}

func LoadDefaultConfigFile() (Config, error) {
	path := DefaultConfigPath()
	cfg, err := LoadConfigFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	return cfg, err
}

func SaveConfigFile(path string, cfg Config) error {
	if cfg.Backend == "" && cfg.APIURL != "" && cfg.APIKey != "" {
		cfg.Backend = "http"
	}
	raw := fileConfig{
		Backend:    cfg.Backend,
		SvcBaseURL: cfg.APIURL,
		APIKey:     cfg.APIKey,
	}
	if cfg.Timeout > 0 {
		raw.TimeoutSeconds = int(cfg.Timeout / time.Second)
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func CurrentConfig() Config {
	return config
}

// ColumnNames returns just the column name strings.
func (r *QueryResult) ColumnNames() []string {
	names := make([]string, len(r.Columns))
	for i, c := range r.Columns {
		names[i] = c.Name
	}
	return names
}

// Query executes a SQL query against the HTTP API and returns the raw result.
func Query(database, sql string) (*QueryResult, error) {
	return queryHTTP(database, sql)
}

func queryHTTP(database, sql string) (*QueryResult, error) {
	if config.APIURL == "" {
		return nil, fmt.Errorf("svc_base_url is required (run: dewbu config set --svc-base-url ...)")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("api_key is required (run: dewbu config set --api-key ...)")
	}
	endpoint := queryEndpoint(config.APIURL)
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

func queryEndpoint(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(trimmed, "/v1/query") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/api/cli") {
		return trimmed + "/v1/query"
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && (parsed.Path == "" || parsed.Path == "/") {
		return trimmed + "/api/cli/v1/query"
	}
	return trimmed + "/v1/query"
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

// apiBase resolves the API root (".../api/cli") from the configured base URL,
// accepting either a service base, an api/cli base, or a full /v1/* endpoint.
func apiBase(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if idx := strings.Index(trimmed, "/api/cli"); idx >= 0 {
		return trimmed[:idx] + "/api/cli"
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && (parsed.Path == "" || parsed.Path == "/") {
		return trimmed + "/api/cli"
	}
	return trimmed + "/api/cli"
}

// APIError carries the HTTP status so callers can give role-aware messages
// (e.g. 401 → check key, 403 → needs admin key).
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return e.Message }

// Request performs an authenticated JSON request against an /api/cli/v1 path
// (e.g. "/v1/personas?brand=dewbu") and decodes the response into out (may be nil).
// Used by the persona management commands.
func Request(method, path string, body interface{}, out interface{}) error {
	if config.APIURL == "" {
		return fmt.Errorf("svc_base_url is required (run: dewbu config set --svc-base-url ...)")
	}
	if config.APIKey == "" {
		return fmt.Errorf("api_key is required (run: dewbu config set --api-key ...)")
	}

	endpoint := apiBase(config.APIURL) + path

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		var errBody struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &errBody) == nil && errBody.Error != "" {
			msg = errBody.Error
			if errBody.Message != "" {
				msg = errBody.Error + ": " + errBody.Message
			}
		}
		if msg == "" {
			msg = fmt.Sprintf("request returned status %d", resp.StatusCode)
		}
		return &APIError{Status: resp.StatusCode, Message: msg}
	}

	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
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

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
