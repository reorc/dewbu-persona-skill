package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/reorc/dewbu-persona-skill/internal/filter"
	"github.com/reorc/dewbu-persona-skill/internal/model"
	"github.com/spf13/cobra"
)
var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Query user profiles",
}

var profileSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search user profiles with filters",
	RunE:  runProfileSearch,
}

var profileSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Show available filter fields and tag enumerations",
	RunE:  runProfileSchema,
}

var (
	prFilterJSON string
	prSource     string
	prSpendMin   float64
	prSpendMax   float64
	prOrderMin   int
	prTag        []string
	prDimension  string
	prQuery      string
	// Dimension-level ILIKE flags
	prPainPoints   string
	prStrengths    string
	prUseCases     string
	prPurchaseMot  string
	prOccupations  string
	prDemographic  string
	prProdInterest string
	prCustStage    string
	prContactInt   string
	prCommValue    string
)

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileSearchCmd)
	profileCmd.AddCommand(profileSchemaCmd)

	profileSearchCmd.Flags().StringVar(&prFilterJSON, "filter", "", "JSON filter DSL")
	profileSearchCmd.Flags().StringVar(&prSource, "source", "", "filter by source_type in source_types array")
	profileSearchCmd.Flags().Float64Var(&prSpendMin, "spend-min", 0, "minimum total_spend")
	profileSearchCmd.Flags().Float64Var(&prSpendMax, "spend-max", 0, "maximum total_spend")
	profileSearchCmd.Flags().IntVar(&prOrderMin, "order-min", 0, "minimum order_count")
	profileSearchCmd.Flags().StringSliceVar(&prTag, "tag", nil, "filter by std tag exact match (repeatable)")
	profileSearchCmd.Flags().StringVar(&prQuery, "query", "", "smart search: ILIKE across all std_* columns")
	// Dimension-level ILIKE flags
	profileSearchCmd.Flags().StringVar(&prPainPoints, "pain-points", "", "ILIKE search in std_pain_points")
	profileSearchCmd.Flags().StringVar(&prStrengths, "strengths", "", "ILIKE search in std_strengths")
	profileSearchCmd.Flags().StringVar(&prUseCases, "use-cases", "", "ILIKE search in std_use_cases")
	profileSearchCmd.Flags().StringVar(&prPurchaseMot, "purchase-motivations", "", "ILIKE search in std_purchase_motivations")
	profileSearchCmd.Flags().StringVar(&prOccupations, "occupations", "", "ILIKE search in std_occupations")
	profileSearchCmd.Flags().StringVar(&prDemographic, "demographic", "", "ILIKE search in std_demographic_signals")
	profileSearchCmd.Flags().StringVar(&prProdInterest, "product-interests", "", "ILIKE search in std_product_interests")
	profileSearchCmd.Flags().StringVar(&prCustStage, "customer-stage", "", "ILIKE search in std_customer_stage")
	profileSearchCmd.Flags().StringVar(&prContactInt, "contact-intents", "", "ILIKE search in std_contact_intents")
	profileSearchCmd.Flags().StringVar(&prCommValue, "commercial-value", "", "ILIKE search in std_commercial_value_signals")

	profileSchemaCmd.Flags().StringVar(&prDimension, "dimension", "", "specific dimension to list tags for")
}

func runProfileSearch(cmd *cobra.Command, args []string) error {
	f, err := buildProfileFilter()
	if err != nil {
		return err
	}

	where := f.ToProfileSQL()

	// Add source_types filter if specified via flag
	if prSource != "" && prFilterJSON == "" {
		extra := "source_types && " + db.EscapeArray([]string{prSource})
		if where == "TRUE" {
			where = extra
		} else {
			where = where + " AND " + extra
		}
	}

	// Add dimension-level ILIKE filters
	dimClauses := buildProfileDimensionClauses()
	if dimClauses != "" {
		if where == "TRUE" {
			where = dimClauses
		} else {
			where = where + " AND " + dimClauses
		}
	}

	// Add --query: cross-dimension ILIKE search
	if prQuery != "" && prFilterJSON == "" {
		smartWhere := filter.SmartQuerySQLNoMatchedTags(prQuery)
		if where == "TRUE" {
			where = smartWhere
		} else {
			where = where + " AND " + smartWhere
		}
	}

	countSQL := fmt.Sprintf("SELECT count(*) FROM user_profiles WHERE %s", where)
	total, err := db.QueryScalar(database, countSQL)
	if err != nil {
		return fmt.Errorf("count query failed: %w", err)
	}
	totalInt := parseCount(total)

	selectFields := "user_id, source_types, is_merged_cross_channel, order_count, total_spend, first_seen_at, last_seen_at, std_pain_points, std_strengths, std_use_cases, std_occupations, std_product_interests, std_customer_stage, countries, product_names, inferred_gender, inferred_age_range"
	querySQL := fmt.Sprintf(
		"SELECT %s FROM user_profiles WHERE %s ORDER BY total_spend DESC NULLS LAST LIMIT %d OFFSET %d",
		selectFields, where, limit, offset,
	)

	rows, err := db.QueryRows(database, querySQL)
	if err != nil {
		return fmt.Errorf("search query failed: %w", err)
	}

	resp := &model.Response{
		Meta: model.Meta{
			Command:  "profile.search",
			Database: database,
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

func buildProfileDimensionClauses() string {
	var clauses []string
	dimFlags := []struct {
		col string
		val string
	}{
		{"std_pain_points", prPainPoints},
		{"std_strengths", prStrengths},
		{"std_use_cases", prUseCases},
		{"std_purchase_motivations", prPurchaseMot},
		{"std_occupations", prOccupations},
		{"std_demographic_signals", prDemographic},
		{"std_product_interests", prProdInterest},
		{"std_customer_stage", prCustStage},
		{"std_contact_intents", prContactInt},
		{"std_commercial_value_signals", prCommValue},
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

func runProfileSchema(cmd *cobra.Command, args []string) error {
	if prDimension != "" {
		// Return tags for a specific dimension
		sql := fmt.Sprintf(
			"SELECT tag_value, evidence_count, user_count FROM tag_dictionary WHERE dimension = %s ORDER BY evidence_count DESC",
			db.EscapeString(prDimension),
		)
		rows, err := db.QueryRows(database, sql)
		if err != nil {
			return fmt.Errorf("schema query failed: %w", err)
		}

		resp := &model.Response{
			Meta: model.Meta{
				Command:  "profile.schema",
				Database: database,
				Total:    len(rows),
				Returned: len(rows),
				Limit:    len(rows),
				Offset:   0,
			},
			Data: map[string]interface{}{
				"dimension": prDimension,
				"tags":      rows,
			},
		}
		return outputResponse(resp)
	}

	// Return full schema overview
	dimensionSQL := `SELECT dimension, count(*) as tag_count, sum(evidence_count) as total_mentions FROM tag_dictionary GROUP BY dimension ORDER BY total_mentions DESC`
	dimensions, err := db.QueryRows(database, dimensionSQL)
	if err != nil {
		return fmt.Errorf("schema query failed: %w", err)
	}

	filterFields := []map[string]string{
		{"field": "source_type", "type": "enum", "values": "amazon_review, email, shopify_order, shopify_review"},
		{"field": "star", "type": "range", "values": "1-5"},
		{"field": "country", "type": "enum", "values": "query evidence_index"},
		{"field": "brand", "type": "string", "values": "DEWBU, ORORO, Wulcea, Venustas"},
		{"field": "asin", "type": "string", "values": "Amazon ASIN"},
		{"field": "time", "type": "range", "values": "after/before ISO dates"},
		{"field": "spend", "type": "range", "values": "gte/lte numeric"},
		{"field": "order_count", "type": "range", "values": "gte/lte integer"},
		{"field": "user_id", "type": "string", "values": "exact match"},
	}

	schema := map[string]interface{}{
		"filter_fields": filterFields,
		"dimensions":    dimensions,
	}

	out, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Println(string(out))
	return nil
}

func buildProfileFilter() (*filter.Filter, error) {
	if prFilterJSON != "" {
		return filter.Parse(prFilterJSON)
	}

	f := &filter.Filter{}
	if prSpendMin > 0 {
		f.Spend = &filter.RangeOp{Gte: &prSpendMin}
	}
	if prSpendMax > 0 {
		if f.Spend == nil {
			f.Spend = &filter.RangeOp{}
		}
		f.Spend.Lte = &prSpendMax
	}
	if prOrderMin > 0 {
		v := float64(prOrderMin)
		f.OrderCount = &filter.RangeOp{Gte: &v}
	}
	if len(prTag) > 0 {
		f.Tags = &filter.ArrayOp{Any: prTag}
	}
	return f, nil
}
