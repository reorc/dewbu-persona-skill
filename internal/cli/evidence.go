package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/reorc/dewbu-persona-skill/internal/filter"
	"github.com/reorc/dewbu-persona-skill/internal/model"
	"github.com/spf13/cobra"
)

var evidenceCmd = &cobra.Command{
	Use:   "evidence",
	Short: "Query evidence records",
}

var evidenceSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search evidence with filters",
	RunE:  runEvidenceSearch,
}

var evidenceGetCmd = &cobra.Command{
	Use:   "get [evidence_id]",
	Short: "Get a single evidence record with full content",
	Args:  cobra.ExactArgs(1),
	RunE:  runEvidenceGet,
}

// CLI flags for evidence search
var (
	evFilterJSON string
	evSource     string
	evStarMin    int
	evStarMax    int
	evTag        []string
	evCountry    string
	evUser       string
	evQuery      string
	// Dimension-level ILIKE flags
	evPainPoints   string
	evStrengths    string
	evUseCases     string
	evPurchaseMot  string
	evOccupations  string
	evDemographic  string
	evProdInterest string
	evCustStage    string
	evContactInt   string
	evCommValue    string
)

func init() {
	rootCmd.AddCommand(evidenceCmd)
	evidenceCmd.AddCommand(evidenceSearchCmd)
	evidenceCmd.AddCommand(evidenceGetCmd)

	evidenceSearchCmd.Flags().StringVar(&evFilterJSON, "filter", "", "JSON filter DSL")
	evidenceSearchCmd.Flags().StringVar(&evSource, "source", "", "source_type filter")
	evidenceSearchCmd.Flags().IntVar(&evStarMin, "star-min", 0, "minimum star rating")
	evidenceSearchCmd.Flags().IntVar(&evStarMax, "star-max", 0, "maximum star rating")
	evidenceSearchCmd.Flags().StringSliceVar(&evTag, "tag", nil, "filter by tag exact match (repeatable)")
	evidenceSearchCmd.Flags().StringVar(&evCountry, "country", "", "country filter")
	evidenceSearchCmd.Flags().StringVar(&evUser, "user", "", "user_id filter")
	evidenceSearchCmd.Flags().StringVar(&evQuery, "query", "", "smart search: tags (ILIKE) + full-text (FTS)")
	// Dimension-level ILIKE flags
	evidenceSearchCmd.Flags().StringVar(&evPainPoints, "pain-points", "", "ILIKE search in pain_points_mapped")
	evidenceSearchCmd.Flags().StringVar(&evStrengths, "strengths", "", "ILIKE search in strengths_mapped")
	evidenceSearchCmd.Flags().StringVar(&evUseCases, "use-cases", "", "ILIKE search in use_cases_mapped")
	evidenceSearchCmd.Flags().StringVar(&evPurchaseMot, "purchase-motivations", "", "ILIKE search in purchase_motivations_mapped")
	evidenceSearchCmd.Flags().StringVar(&evOccupations, "occupations", "", "ILIKE search in occupations_mapped")
	evidenceSearchCmd.Flags().StringVar(&evDemographic, "demographic", "", "ILIKE search in demographic_signals_mapped")
	evidenceSearchCmd.Flags().StringVar(&evProdInterest, "product-interests", "", "ILIKE search in product_interests_mapped")
	evidenceSearchCmd.Flags().StringVar(&evCustStage, "customer-stage", "", "ILIKE search in customer_stage_mapped")
	evidenceSearchCmd.Flags().StringVar(&evContactInt, "contact-intents", "", "ILIKE search in contact_intents_mapped")
	evidenceSearchCmd.Flags().StringVar(&evCommValue, "commercial-value", "", "ILIKE search in commercial_value_signals_mapped")
}

func runEvidenceSearch(cmd *cobra.Command, args []string) error {
	f, err := buildEvidenceFilter()
	if err != nil {
		return err
	}

	tieredQuery := ""
	if evQuery != "" && evFilterJSON == "" {
		tieredQuery = evQuery
		f.Query = ""
	}
	where := f.ToEvidenceSQL()

	// Add dimension-level ILIKE filters
	dimClauses := buildDimensionClauses()
	if dimClauses != "" {
		if where == "TRUE" {
			where = dimClauses
		} else {
			where = where + " AND " + dimClauses
		}
	}

	// If --query is set, use tiered search (tag ILIKE + FTS)
	if tieredQuery != "" {
		return runTieredSearch(where, tieredQuery)
	}

	// Standard search (no --query, or --query via JSON filter which uses FTS only)
	countSQL := fmt.Sprintf("SELECT count(*) FROM evidence_index WHERE %s", where)
	total, err := db.QueryScalar(countSQL)
	if err != nil {
		return fmt.Errorf("count query failed: %w", err)
	}
	totalInt := parseCount(total)

	selectFields := "evidence_id, source_type, user_id, platform, brand, country, event_time, star, title, content_snippet, pain_points_mapped, strengths_mapped, use_cases_mapped, extraction_confidence"
	orderBy := "event_time DESC NULLS LAST"
	if f.Query != "" {
		query := db.EscapeString(f.Query)
		selectFields += fmt.Sprintf(", ts_rank(content_tsv, plainto_tsquery('english', %s)) AS relevance", query)
		orderBy = "relevance DESC NULLS LAST, event_time DESC NULLS LAST"
	}
	querySQL := fmt.Sprintf(
		"SELECT %s FROM evidence_index WHERE %s ORDER BY %s LIMIT %d OFFSET %d",
		selectFields, where, orderBy, limit, offset,
	)

	rows, err := db.QueryRows(querySQL)
	if err != nil {
		return fmt.Errorf("search query failed: %w", err)
	}

	resp := &model.Response{
		Meta: model.Meta{
			Command:  "evidence.search",
			Filter:   filterDescription(f),
			Total:    totalInt,
			Returned: len(rows),
			Limit:    limit,
			Offset:   offset,
		},
		Data: rows,
	}

	return outputResponse(resp)
}

// runTieredSearch implements the two-tier search strategy:
// Tier 1: ILIKE across all std_* columns and matched_tags
// Tier 2: FTS on content_text (excluding tier 1 hits)
func runTieredSearch(baseWhere, query string) error {
	smartWhere := filter.SmartQuerySQL(query)
	ftsQuery := db.EscapeString(query)

	// Count tier 1 (tag hits)
	tier1CountSQL := fmt.Sprintf(
		"SELECT count(*) FROM evidence_index WHERE %s AND %s",
		baseWhere, smartWhere,
	)
	tier1Count, _ := db.QueryScalar(tier1CountSQL)
	tier1Int := parseCount(tier1Count)

	// Count tier 2 (FTS hits not in tier 1)
	tier2CountSQL := fmt.Sprintf(
		"SELECT count(*) FROM evidence_index WHERE %s AND content_tsv @@ plainto_tsquery('english', %s) AND NOT %s",
		baseWhere, ftsQuery, smartWhere,
	)
	tier2Count, _ := db.QueryScalar(tier2CountSQL)
	tier2Int := parseCount(tier2Count)

	totalInt := tier1Int + tier2Int

	// Build tiered query with CTE
	tieredSQL := fmt.Sprintf(`WITH tag_hits AS (
    SELECT evidence_id, 1 as match_tier, 0::float as fts_rank
    FROM evidence_index
    WHERE %s AND %s
),
fts_hits AS (
    SELECT evidence_id, 2 as match_tier,
           ts_rank(content_tsv, plainto_tsquery('english', %s)) as fts_rank
    FROM evidence_index
    WHERE %s AND content_tsv @@ plainto_tsquery('english', %s)
      AND evidence_id NOT IN (SELECT evidence_id FROM tag_hits)
),
combined AS (
    SELECT * FROM tag_hits
    UNION ALL
    SELECT * FROM fts_hits
)
SELECT e.evidence_id, e.source_type, e.user_id, e.platform, e.brand, e.country,
       e.event_time, e.star, e.title, e.content_snippet,
       e.pain_points_mapped, e.strengths_mapped, e.use_cases_mapped,
       e.extraction_confidence, c.match_tier, c.fts_rank
FROM combined c
JOIN evidence_index e USING (evidence_id)
ORDER BY c.match_tier ASC, c.fts_rank DESC NULLS LAST, e.event_time DESC NULLS LAST
LIMIT %d OFFSET %d`,
		baseWhere, smartWhere,
		ftsQuery,
		baseWhere, ftsQuery,
		limit, offset,
	)

	rows, err := db.QueryRows(tieredSQL)
	if err != nil {
		return fmt.Errorf("tiered search failed: %w", err)
	}

	resp := &model.Response{
		Meta: model.Meta{
			Command: "evidence.search",
			Filter: map[string]interface{}{
				"query":       query,
				"mode":        "tiered",
				"tier1_count": tier1Int,
				"tier2_count": tier2Int,
			},
			Total:    totalInt,
			Returned: len(rows),
			Limit:    limit,
			Offset:   offset,
		},
		Data: rows,
	}

	return outputResponse(resp)
}

func buildDimensionClauses() string {
	var clauses []string
	dimFlags := []struct {
		col string
		val string
	}{
		{"pain_points_mapped", evPainPoints},
		{"strengths_mapped", evStrengths},
		{"use_cases_mapped", evUseCases},
		{"purchase_motivations_mapped", evPurchaseMot},
		{"occupations_mapped", evOccupations},
		{"demographic_signals_mapped", evDemographic},
	}
	for _, d := range dimFlags {
		if d.val != "" {
			clauses = append(clauses, filter.DimensionLikeSQL(d.col, d.val))
		}
	}
	if len(clauses) == 0 {
		return ""
	}
	return strings.Join(clauses, " AND ")
}

func parseCount(val interface{}) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case string:
		var n int
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	return 0
}

func runEvidenceGet(cmd *cobra.Command, args []string) error {
	evidenceID := args[0]

	querySQL := fmt.Sprintf(
		`SELECT e.*, COALESCE(
			(SELECT a.content FROM amazon_review_signals a WHERE a.signal_record_id = e.signal_record_id),
			(SELECT s.message_content FROM email_signals s WHERE s.signal_record_id = e.signal_record_id),
			(SELECT sr.content FROM shopify_review_signals sr WHERE sr.signal_record_id = e.signal_record_id)
		) as full_content
		FROM evidence_index e WHERE e.evidence_id = %s`,
		db.EscapeString(evidenceID),
	)

	rows, err := db.QueryRows(querySQL)
	if err != nil {
		return fmt.Errorf("get query failed: %w", err)
	}

	if len(rows) == 0 {
		return fmt.Errorf("evidence not found: %s", evidenceID)
	}

	resp := &model.Response{
		Meta: model.Meta{
			Command:  "evidence.get",
			Total:    1,
			Returned: 1,
			Limit:    1,
			Offset:   0,
		},
		Data: rows[0],
	}

	return outputResponse(resp)
}

func buildEvidenceFilter() (*filter.Filter, error) {
	if evFilterJSON != "" {
		return filter.Parse(evFilterJSON)
	}

	// Build filter from CLI flags
	f := &filter.Filter{}
	if evSource != "" {
		f.SourceType = evSource
	}
	if evStarMin > 0 || evStarMax > 0 {
		starMap := map[string]interface{}{}
		if evStarMin > 0 {
			starMap["gte"] = float64(evStarMin)
		}
		if evStarMax > 0 {
			starMap["lte"] = float64(evStarMax)
		}
		f.Star = starMap
	}
	if len(evTag) > 0 {
		f.Tags = &filter.ArrayOp{Any: evTag}
	}
	if evCountry != "" {
		f.Country = evCountry
	}
	if evUser != "" {
		f.UserID = evUser
	}
	if evQuery != "" {
		f.Query = evQuery
	}
	return f, nil
}

func filterDescription(f *filter.Filter) interface{} {
	// Return the filter as a map for JSON output
	b, _ := json.Marshal(f)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	// Remove nil/empty fields
	clean := make(map[string]interface{})
	for k, v := range m {
		if v != nil && v != "" {
			clean[k] = v
		}
	}
	if len(clean) == 0 {
		return nil
	}
	return clean
}

func outputResponse(resp *model.Response) error {
	switch strings.ToLower(format) {
	case "json":
		out, err := resp.Marshal()
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	case "table":
		// Simple table output for human readability
		out, _ := json.MarshalIndent(resp.Data, "", "  ")
		fmt.Println(string(out))
	default:
		out, err := resp.Marshal()
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(out))
	}
	return nil
}
