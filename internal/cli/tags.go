package cli

import (
	"fmt"
	"strings"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/reorc/dewbu-persona-skill/internal/filter"
	"github.com/reorc/dewbu-persona-skill/internal/model"
	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Explore and search tags",
}

var tagsSearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search tags across all dimensions by keyword (substring match)",
	Args:  cobra.ExactArgs(1),
	RunE:  runTagsSearch,
}

func init() {
	rootCmd.AddCommand(tagsCmd)
	tagsCmd.AddCommand(tagsSearchCmd)
}

func runTagsSearch(cmd *cobra.Command, args []string) error {
	keyword := args[0]
	escaped := db.EscapeString("%" + keyword + "%")

	// Part 1: Search tag_dictionary
	dictSQL := fmt.Sprintf(
		"SELECT dimension, tag_value, evidence_count, user_count FROM tag_dictionary WHERE tag_value ILIKE %s ORDER BY evidence_count DESC LIMIT 50",
		escaped,
	)
	dictRows, err := db.QueryRows(database, dictSQL)
	if err != nil {
		return fmt.Errorf("tag_dictionary search failed: %w", err)
	}

	// Part 2: Search actual values in evidence_index *_mapped columns
	cols := filter.StdColumns()
	var unions []string
	for _, col := range cols {
		unions = append(unions, fmt.Sprintf(
			"SELECT '%s' as dimension, _t as tag, count(*) as evidence_count FROM evidence_index, unnest(%s) _t WHERE _t ILIKE %s GROUP BY _t",
			col.Dimension, col.Name, escaped,
		))
	}

	usageSQL := fmt.Sprintf(
		"SELECT dimension, tag, evidence_count FROM (%s) sub ORDER BY evidence_count DESC LIMIT %d",
		strings.Join(unions, " UNION ALL "),
		limit,
	)
	usageRows, err := db.QueryRows(database, usageSQL)
	if err != nil {
		return fmt.Errorf("usage search failed: %w", err)
	}

	resp := &model.Response{
		Meta: model.Meta{
			Command:  "tags.search",
			Database: database,
			Total:    len(dictRows) + len(usageRows),
			Returned: len(dictRows) + len(usageRows),
			Limit:    limit,
			Offset:   0,
		},
		Data: map[string]interface{}{
			"keyword":    keyword,
			"dictionary": dictRows,
			"usage":      usageRows,
		},
	}

	return outputResponse(resp)
}
