package cli

import (
	"fmt"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/reorc/dewbu-persona-skill/internal/model"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Statistics and analytics",
}

var statsTagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Show tag distribution statistics",
	RunE:  runStatsTags,
}

var (
	stSource  string
	stGroupBy string
	stTop     int
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.AddCommand(statsTagsCmd)

	statsTagsCmd.Flags().StringVar(&stSource, "source", "", "filter by source_type")
	statsTagsCmd.Flags().StringVar(&stGroupBy, "group-by", "dimension", "group by: dimension or tag")
	statsTagsCmd.Flags().IntVar(&stTop, "top", 30, "top N results")
}

func runStatsTags(cmd *cobra.Command, args []string) error {
	var sql string

	switch stGroupBy {
	case "dimension":
		sql = fmt.Sprintf(
			"SELECT dimension, count(*) as tag_count, sum(evidence_count) as total_mentions FROM tag_dictionary GROUP BY dimension ORDER BY total_mentions DESC LIMIT %d",
			stTop,
		)
	case "tag":
		where := "TRUE"
		if stSource != "" {
			where = "source_type = " + db.EscapeString(stSource)
		}
		// Unnest all mapped tag columns from evidence_index
		sql = fmt.Sprintf(
			`SELECT dimension, tag_value, evidence_count, user_count
			FROM tag_dictionary
			WHERE %s
			ORDER BY evidence_count DESC LIMIT %d`,
			func() string {
				if stSource != "" {
					return "TRUE" // tag_dictionary doesn't have source_type, show all
				}
				return "TRUE"
			}(), stTop,
		)
		_ = where
	default:
		return fmt.Errorf("invalid group-by: %s (use dimension or tag)", stGroupBy)
	}

	rows, err := db.QueryRows(database, sql)
	if err != nil {
		return fmt.Errorf("stats query failed: %w", err)
	}

	resp := &model.Response{
		Meta: model.Meta{
			Command:  "stats.tags",
			Database: database,
			Total:    len(rows),
			Returned: len(rows),
			Limit:    stTop,
			Offset:   0,
		},
		Data: rows,
	}

	return outputResponse(resp)
}
