package db

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPBackendQueryRowsAndScalar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/query" {
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
