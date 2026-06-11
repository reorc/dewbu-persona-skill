package db

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHTTPBackendQueryRowsAndScalar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/v1/query" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["database"] != "dewbu_persona_v2" {
			t.Fatalf("unexpected database: %s", req["database"])
		}
		_ = json.NewEncoder(w).Encode(QueryResult{
			Columns:  []Column{{Name: "count", Type: "int8"}},
			Rows:     [][]interface{}{{float64(42)}},
			RowCount: 1,
			Command:  "sql",
		})
	}))
	defer server.Close()

	oldConfig := config
	defer func() { config = oldConfig }()
	Configure(Config{
		Backend: "http",
		APIURL:  server.URL,
		APIKey:  "test-key",
		Timeout: time.Second,
	})

	rows, err := QueryRows("dewbu_persona_v2", "SELECT count(*) FROM evidence_index")
	if err != nil {
		t.Fatal(err)
	}
	if got := rows[0]["count"].(float64); got != 42 {
		t.Fatalf("unexpected row value: %v", got)
	}
	scalar, err := QueryScalar("dewbu_persona_v2", "SELECT count(*) FROM evidence_index")
	if err != nil {
		t.Fatal(err)
	}
	if got := scalar.(float64); got != 42 {
		t.Fatalf("unexpected scalar: %v", got)
	}
}

func TestLoadConfigFileInfersHTTPBackend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "svc_base_url": "https://example.test/api/cli",
  "api_key": "dewbu_live_test",
  "timeout_seconds": 12
}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backend != "http" {
		t.Fatalf("expected inferred http backend, got %q", cfg.Backend)
	}
	if cfg.APIURL != "https://example.test/api/cli" {
		t.Fatalf("unexpected api url: %q", cfg.APIURL)
	}
	if cfg.APIKey != "dewbu_live_test" {
		t.Fatalf("unexpected api key: %q", cfg.APIKey)
	}
	if cfg.Timeout != 12*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.Timeout)
	}
}

func TestQueryEndpointAcceptsServiceBaseOrFullEndpoint(t *testing.T) {
	tests := map[string]string{
		"https://example.test":                  "https://example.test/api/cli/v1/query",
		"https://example.test/":                 "https://example.test/api/cli/v1/query",
		"https://example.test/api/cli":          "https://example.test/api/cli/v1/query",
		"https://example.test/api/cli/":         "https://example.test/api/cli/v1/query",
		"https://example.test/api/cli/v1/query": "https://example.test/api/cli/v1/query",
		"https://example.test/custom":           "https://example.test/custom/v1/query",
	}
	for input, want := range tests {
		if got := queryEndpoint(input); got != want {
			t.Fatalf("queryEndpoint(%q) = %q, want %q", input, got, want)
		}
	}
}
