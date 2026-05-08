package filter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/reorc/dewbu-persona-skill/internal/db"
)

// Filter represents the parsed filter DSL.
type Filter struct {
	SourceType  interface{} `json:"source_type"`  // string or []string
	Star        interface{} `json:"star"`         // int or {"gte":N,"lte":N}
	Country     interface{} `json:"country"`      // string or []string
	Brand       string      `json:"brand"`
	Asin        string      `json:"asin"`
	Time        *TimeRange  `json:"time"`
	Tags        *ArrayOp    `json:"tags"`
	PainPoints  *ArrayOp    `json:"pain_points"`
	Strengths   *ArrayOp    `json:"strengths"`
	UseCases    *ArrayOp    `json:"use_cases"`
	PurchaseMot *ArrayOp    `json:"purchase_motivations"`
	Occupations *ArrayOp    `json:"occupations"`
	Demographic *ArrayOp    `json:"demographic_signals"`
	ProdInterest *ArrayOp   `json:"product_interests"`
	CustStage   *ArrayOp    `json:"customer_stage"`
	ContactInt  *ArrayOp    `json:"contact_intents"`
	CommValue   *ArrayOp    `json:"commercial_value_signals"`
	Spend       *RangeOp    `json:"spend"`
	OrderCount  *RangeOp    `json:"order_count"`
	UserID      string      `json:"user_id"`
	Query       string      `json:"query"`
}

type TimeRange struct {
	After  string `json:"after"`
	Before string `json:"before"`
}

type ArrayOp struct {
	Any []string `json:"any"`
	All []string `json:"all"`
}

type RangeOp struct {
	Gte *float64 `json:"gte"`
	Lte *float64 `json:"lte"`
}

// Parse parses a JSON filter string into a Filter struct.
func Parse(filterJSON string) (*Filter, error) {
	if filterJSON == "" {
		return &Filter{}, nil
	}
	var f Filter
	if err := json.Unmarshal([]byte(filterJSON), &f); err != nil {
		return nil, fmt.Errorf("invalid filter JSON: %w", err)
	}
	return &f, nil
}

// ToSQL converts a Filter to SQL WHERE clauses for evidence_index.
func (f *Filter) ToEvidenceSQL() string {
	clauses := f.commonClauses()

	// star
	if f.Star != nil {
		clauses = append(clauses, rangeOrExact("star", f.Star)...)
	}

	// asin
	if f.Asin != "" {
		clauses = append(clauses, "asin = "+db.EscapeString(f.Asin))
	}

	// tags - removed in v2, tags are now in dimension-specific *_mapped columns
	// if f.Tags != nil {
	// 	clauses = append(clauses, arrayOpSQL("matched_tags", f.Tags)...)
	// }

	// *_mapped columns
	clauses = append(clauses, stdArrayClauses(f)...)

	// query (FTS)
	if f.Query != "" {
		clauses = append(clauses, "content_tsv @@ plainto_tsquery('english', "+db.EscapeString(f.Query)+")")
	}

	if len(clauses) == 0 {
		return "TRUE"
	}
	return strings.Join(clauses, " AND ")
}

// ToProfileSQL converts a Filter to SQL WHERE clauses for user_profiles.
func (f *Filter) ToProfileSQL() string {
	clauses := f.commonClauses()

	// spend
	if f.Spend != nil {
		if f.Spend.Gte != nil {
			clauses = append(clauses, fmt.Sprintf("total_spend >= %v", *f.Spend.Gte))
		}
		if f.Spend.Lte != nil {
			clauses = append(clauses, fmt.Sprintf("total_spend <= %v", *f.Spend.Lte))
		}
	}

	// order_count
	if f.OrderCount != nil {
		if f.OrderCount.Gte != nil {
			clauses = append(clauses, fmt.Sprintf("order_count >= %v", int(*f.OrderCount.Gte)))
		}
		if f.OrderCount.Lte != nil {
			clauses = append(clauses, fmt.Sprintf("order_count <= %v", int(*f.OrderCount.Lte)))
		}
	}

	// tags on profiles use std_* columns directly
	clauses = append(clauses, stdArrayClauses(f)...)

	if len(clauses) == 0 {
		return "TRUE"
	}
	return strings.Join(clauses, " AND ")
}

func (f *Filter) commonClauses() []string {
	var clauses []string

	// source_type
	if f.SourceType != nil {
		switch v := f.SourceType.(type) {
		case string:
			clauses = append(clauses, "source_type = "+db.EscapeString(v))
		case []interface{}:
			vals := make([]string, len(v))
			for i, item := range v {
				vals[i] = db.EscapeString(fmt.Sprint(item))
			}
			clauses = append(clauses, "source_type IN ("+strings.Join(vals, ",")+")")
		}
	}

	// country
	if f.Country != nil {
		switch v := f.Country.(type) {
		case string:
			clauses = append(clauses, "country = "+db.EscapeString(v))
		case []interface{}:
			vals := make([]string, len(v))
			for i, item := range v {
				vals[i] = db.EscapeString(fmt.Sprint(item))
			}
			clauses = append(clauses, "country IN ("+strings.Join(vals, ",")+")")
		}
	}

	// brand
	if f.Brand != "" {
		clauses = append(clauses, "brand = "+db.EscapeString(f.Brand))
	}

	// time
	if f.Time != nil {
		if f.Time.After != "" {
			clauses = append(clauses, "event_time >= "+db.EscapeString(f.Time.After)+"::timestamptz")
		}
		if f.Time.Before != "" {
			clauses = append(clauses, "event_time <= "+db.EscapeString(f.Time.Before)+"::timestamptz")
		}
	}

	// user_id
	if f.UserID != "" {
		clauses = append(clauses, "user_id = "+db.EscapeString(f.UserID))
	}

	return clauses
}

func stdArrayClauses(f *Filter) []string {
	var clauses []string
	pairs := []struct {
		col string
		op  *ArrayOp
	}{
		{"pain_points_mapped", f.PainPoints},
		{"strengths_mapped", f.Strengths},
		{"use_cases_mapped", f.UseCases},
		{"purchase_motivations_mapped", f.PurchaseMot},
		{"occupations_mapped", f.Occupations},
		{"demographic_signals_mapped", f.Demographic},
	}
	for _, p := range pairs {
		if p.op != nil {
			clauses = append(clauses, arrayOpSQL(p.col, p.op)...)
		}
	}
	return clauses
}

func arrayOpSQL(col string, op *ArrayOp) []string {
	var clauses []string
	if len(op.Any) > 0 {
		clauses = append(clauses, col+" && "+db.EscapeArray(op.Any))
	}
	if len(op.All) > 0 {
		clauses = append(clauses, col+" @> "+db.EscapeArray(op.All))
	}
	return clauses
}

func rangeOrExact(col string, val interface{}) []string {
	var clauses []string
	switch v := val.(type) {
	case float64:
		clauses = append(clauses, fmt.Sprintf("%s = %d", col, int(v)))
	case map[string]interface{}:
		if gte, ok := v["gte"]; ok {
			clauses = append(clauses, fmt.Sprintf("%s >= %v", col, int(gte.(float64))))
		}
		if lte, ok := v["lte"]; ok {
			clauses = append(clauses, fmt.Sprintf("%s <= %v", col, int(lte.(float64))))
		}
	}
	return clauses
}

// stdColumns lists all *_mapped array columns on evidence_index.
// Note: evidence_index only has 6 mapped columns; user_profiles has more via std_* columns.
var stdColumns = []struct {
	Name      string // SQL column name
	Dimension string // human-readable dimension label
}{
	{"pain_points_mapped", "pain_points"},
	{"strengths_mapped", "strengths"},
	{"use_cases_mapped", "use_cases"},
	{"purchase_motivations_mapped", "purchase_motivations"},
	{"occupations_mapped", "occupations"},
	{"demographic_signals_mapped", "demographic_signals"},
}

// SmartQuerySQL generates a SQL fragment that matches a keyword across all
// *_mapped array columns using ILIKE substring matching.
// This is the "Tier 1" of the tiered search strategy.
func SmartQuerySQL(keyword string) string {
	return smartQuerySQL(keyword, false) // v2: no matched_tags column
}

// SmartQuerySQLNoMatchedTags generates ILIKE across user_profiles std_* columns.
func SmartQuerySQLNoMatchedTags(keyword string) string {
	escaped := db.EscapeString("%" + keyword + "%")
	profileCols := []string{
		"std_pain_points", "std_strengths", "std_use_cases",
		"std_purchase_motivations", "std_occupations", "std_demographic_signals",
		"std_product_interests", "std_customer_stage", "std_contact_intents",
		"std_commercial_value_signals",
	}
	var parts []string
	for _, col := range profileCols {
		parts = append(parts, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM unnest(%s) _t WHERE _t ILIKE %s)", col, escaped))
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func smartQuerySQL(keyword string, includeMatchedTags bool) string {
	escaped := db.EscapeString("%" + keyword + "%")
	var parts []string
	for _, col := range stdColumns {
		parts = append(parts, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM unnest(%s) _t WHERE _t ILIKE %s)", col.Name, escaped))
	}
	// v2: matched_tags column removed, ignore includeMatchedTags parameter
	return "(" + strings.Join(parts, " OR ") + ")"
}

// DimensionLikeSQL generates a SQL fragment for ILIKE matching on a single *_mapped column.
func DimensionLikeSQL(column, keyword string) string {
	escaped := db.EscapeString("%" + keyword + "%")
	return fmt.Sprintf("EXISTS (SELECT 1 FROM unnest(%s) _t WHERE _t ILIKE %s)", column, escaped)
}

// StdColumns returns the list of std column metadata (for use by tags search).
func StdColumns() []struct {
	Name      string
	Dimension string
} {
	return stdColumns
}
