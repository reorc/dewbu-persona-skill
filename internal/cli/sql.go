package cli

import (
	"fmt"
	"strings"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/reorc/dewbu-persona-skill/internal/model"
	"github.com/spf13/cobra"
)

var sqlCmd = &cobra.Command{
	Use:   "sql [query]",
	Short: "Execute a read-only SQL query against the database",
	Long:  "Execute arbitrary SELECT queries. Write operations (INSERT, UPDATE, DELETE, DROP, ALTER, TRUNCATE, CREATE) are rejected.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSQL,
}

func init() {
	rootCmd.AddCommand(sqlCmd)
}

// blockedKeywords are SQL keywords that indicate write operations.
var blockedKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "TRUNCATE",
	"CREATE", "GRANT", "REVOKE", "COPY", "VACUUM", "REINDEX",
}

func runSQL(cmd *cobra.Command, args []string) error {
	query := strings.TrimSpace(args[0])
	if query == "" {
		return fmt.Errorf("empty query")
	}

	// Check for write operations
	upper := strings.ToUpper(query)
	for _, kw := range blockedKeywords {
		// Check if keyword appears as a word boundary (not inside a string literal)
		// Simple heuristic: check if the keyword appears at the start or after whitespace
		if strings.HasPrefix(upper, kw+" ") || strings.HasPrefix(upper, kw+"\n") || strings.HasPrefix(upper, kw+"\t") ||
			strings.Contains(upper, " "+kw+" ") || strings.Contains(upper, "\n"+kw+" ") ||
			strings.Contains(upper, ";"+kw+" ") || strings.Contains(upper, "("+kw+" ") {
			return fmt.Errorf("write operation not allowed: %s is blocked. Only SELECT queries are permitted", kw)
		}
	}

	// Must start with SELECT or WITH (CTE)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("only SELECT (or WITH ... SELECT) queries are allowed")
	}

	rows, err := db.QueryRows(query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	resp := &model.Response{
		Meta: model.Meta{
			Command:  "sql",
			Total:    len(rows),
			Returned: len(rows),
			Limit:    limit,
			Offset:   0,
		},
		Data: rows,
	}

	return outputResponse(resp)
}
