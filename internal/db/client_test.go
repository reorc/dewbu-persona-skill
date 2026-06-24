package db

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
		if _, ok := req["database"]; ok {
			t.Fatalf("client should no longer send a database; got %q", req["database"])
		}
		if req["sql"] == "" {
			t.Fatalf("missing sql in request body")
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

	rows, err := QueryRows("SELECT count(*) FROM evidence_index")
	if err != nil {
		t.Fatal(err)
	}
	if got := rows[0]["count"].(float64); got != 42 {
		t.Fatalf("unexpected row value: %v", got)
	}
	scalar, err := QueryScalar("SELECT count(*) FROM evidence_index")
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

func TestLoadDefaultConfigFileHonorsExplicitVOCConfig(t *testing.T) {
	t.Setenv("VOC_CONFIG", filepath.Join(t.TempDir(), "missing-voc.json"))
	t.Setenv("DEWBU_CONFIG", "")
	cfg, err := LoadDefaultConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIURL != "" {
		t.Fatalf("explicit VOC_CONFIG should not fall back to legacy config, got %q", cfg.APIURL)
	}
}

func TestLoadDefaultConfigFileFallsBackToLegacyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VOC_CONFIG", "")
	t.Setenv("DEWBU_CONFIG", "")

	legacyPath := filepath.Join(home, ".dewbu", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte(`{
  "svc_base_url": "https://legacy.example.test",
  "api_key": "legacy_key"
}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadDefaultConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIURL != "https://legacy.example.test" {
		t.Fatalf("expected legacy config fallback, got %q", cfg.APIURL)
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

func TestApiBaseResolution(t *testing.T) {
	tests := map[string]string{
		"https://example.test":                     "https://example.test/api/cli",
		"https://example.test/":                    "https://example.test/api/cli",
		"https://example.test/api/cli":             "https://example.test/api/cli",
		"https://example.test/api/cli/":            "https://example.test/api/cli",
		"https://example.test/api/cli/v1/query":    "https://example.test/api/cli",
		"https://example.test/api/cli/v1/personas": "https://example.test/api/cli",
	}
	for input, want := range tests {
		if got := apiBase(input); got != want {
			t.Fatalf("apiBase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRequestGETDecodesBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/cli/v1/personas" || r.URL.RawQuery != "" {
			t.Fatalf("unexpected url: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Fatalf("unexpected auth: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"personas": []interface{}{}})
	}))
	defer server.Close()

	oldConfig := config
	defer func() { config = oldConfig }()
	Configure(Config{APIURL: server.URL, APIKey: "k", Timeout: time.Second})

	var out map[string]interface{}
	if err := Request(http.MethodGet, "/v1/personas", nil, &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["personas"]; !ok {
		t.Fatalf("expected personas key, got %v", out)
	}
}

func TestRequestPatchSendsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["name"] != "renamed" {
			t.Fatalf("unexpected body: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"persona": map[string]interface{}{"name": "renamed"}})
	}))
	defer server.Close()

	oldConfig := config
	defer func() { config = oldConfig }()
	Configure(Config{APIURL: server.URL, APIKey: "k", Timeout: time.Second})

	var out map[string]interface{}
	if err := Request(http.MethodPatch, "/v1/personas/abc", map[string]interface{}{"name": "renamed"}, &out); err != nil {
		t.Fatal(err)
	}
}

func TestRequestForbiddenReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "forbidden",
			"message": "this operation requires an admin API key",
		})
	}))
	defer server.Close()

	oldConfig := config
	defer func() { config = oldConfig }()
	Configure(Config{APIURL: server.URL, APIKey: "k", Timeout: time.Second})

	err := Request(http.MethodPost, "/v1/personas", map[string]interface{}{"name": "x"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", apiErr.Status)
	}
	if !strings.Contains(apiErr.Message, "admin") {
		t.Fatalf("expected admin hint in message, got %q", apiErr.Message)
	}
}
