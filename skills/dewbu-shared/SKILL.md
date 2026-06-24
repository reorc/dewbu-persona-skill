---
name: dewbu-shared
version: 1.0.0
description: "Shared Dewbu guidance for CLI usage, data model, query patterns, and conventions. Read this before using any Dewbu skill."
metadata:
  requires:
    bins: ["dewbu"]
  cliHelp: "dewbu --help"
---

# Dewbu Shared Guide

Read this before using any Dewbu skill. It covers the CLI commands, data model, query patterns, and conventions shared across all Dewbu capabilities.

## CLI Commands

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--db` | `dewbu_persona_v2` | Database name |
| `--format` | `json` | Output: json / table / csv |
| `--limit` | `20` | Max rows returned |
| `--offset` | `0` | Pagination offset |
| `--fields` | all | Comma-separated return fields |

### dewbu sql \<query\>

Execute any read-only SQL (SELECT / WITH ... SELECT). Write operations are rejected.

```bash
dewbu sql "SELECT count(*) FROM evidence_index"
dewbu sql "SELECT brand, count(*) FROM amazon_review_signals GROUP BY brand ORDER BY count(*) DESC"
dewbu sql "SELECT tag_value, evidence_count FROM tag_dictionary WHERE dimension = 'pain_points' ORDER BY evidence_count DESC LIMIT 10"
dewbu sql "SELECT count(*) FROM amazon_reviewer_history_signals"
```

### dewbu tags search \<keyword\>

Search tag_dictionary and all `*_mapped` columns for substring matches.

```bash
dewbu tags search battery
dewbu tags search gift
```

Returns: `dictionary` (tag_dictionary matches with dimension + counts) and `usage` (actual values in evidence_index `*_mapped` columns + counts).

### dewbu evidence search

**Search modes:**

| Flag | Searches | Use when |
|------|----------|----------|
| `--query <kw>` | All `*_mapped` cols (ILIKE) + FTS | Unsure which dimension |
| `--pain-points <kw>` | pain_points_mapped only | Searching pain points |
| `--strengths <kw>` | strengths_mapped only | Searching strengths |
| `--use-cases <kw>` | use_cases_mapped only | Searching use cases |
| `--purchase-motivations <kw>` | purchase_motivations_mapped | Purchase motivations |
| `--occupations <kw>` | occupations_mapped | Occupations |
| `--demographic <kw>` | demographic_signals_mapped | Demographics |

**Structural filters (combinable, AND logic):**

| Flag | Description |
|------|-------------|
| `--source` | amazon_review / email / shopify_order / shopify_review |
| `--star-min` / `--star-max` | Star rating range (1-5) |
| `--country` | Country |
| `--user` | Exact user ID |

**Output:** `match_tier=1` = tag hit (precise), `match_tier=2` = FTS full-text hit.

### dewbu evidence get \<evidence_id\>

Get full evidence detail including original text.

```bash
dewbu evidence get "evidence::amazon_review::review::R1IF9O7VQVGA6G"
```

### dewbu profile search

Same dimension flags as evidence search, but searches `std_*` columns on user_profiles.

**Extra filters:**

| Flag | Description |
|------|-------------|
| `--source` | Filter by source_types array |
| `--spend-min` / `--spend-max` | Spend range |
| `--order-min` | Minimum order count |

```bash
dewbu profile search --query battery --limit 10
dewbu profile search --spend-min 200 --pain-points battery
dewbu profile search --occupations hunter --limit 20
```

### dewbu stats tags

Tag distribution statistics.

```bash
dewbu stats tags --group-by dimension
dewbu stats tags --group-by tag --top 20
```

### dewbu persona (manage saved personas)

Manage saved personas. **Permissions follow your API key**: a read-only (user)
key can `list` / `get`; create / update / delete / build require an **admin**
key. A read-only key attempting a write returns `403 — needs an admin API key`.

```bash
# read-only (any key)
dewbu persona list --brand dewbu
dewbu persona get <persona_id>

# writes (admin key only)
dewbu persona create --brand dewbu --name "Hunters" \
  --description "Cold-weather hunting users" \
  --filter '{"occupations":["hunter"],"stars":[1,2]}'
dewbu persona update <persona_id> --name "Hunters v2" --filter '{"stars":[1,2,3]}'
dewbu persona build  <persona_id>   # recompute cached profile/stats
dewbu persona delete <persona_id>
```

`--filter` is a persona filter config JSON object (same shape the webapp uses):
`brands`, `platforms`, `stars`, `countries`, `timeStart`, `timeEnd`, `tags`
(`[{"dimension","values"}]`), `gender`, `ageRange`, `spendRange`,
`orderCountRange`, `sourceTypes`. Editing the filter clears the cached profile —
run `dewbu persona build <id>` afterwards to refresh stats.

---

## Data Model

**Database:** `dewbu_persona_v2`

Use the `dewbu` CLI for queries. The CLI uses the Dewbu HTTP API by default, with credentials stored in `~/.dewbu/config.json`. Do not call the database backend directly for Dewbu queries.

Onboarding:

1. Open `https://dewbu-persona.tool.reorc.cloud/`
2. Go to `Admin` -> `Accounts`
3. Open `More` -> `API keys` for any active account (the key inherits that
   account's permissions: admin = full persona management, user = read-only)
4. Generate a key, copy it once, then configure the CLI

```bash
dewbu config set --svc-base-url https://dewbu-persona.tool.reorc.cloud/ --api-key dewbu_live_xxx
dewbu config show
```

```
source tables (spoke)          evidence_index (serving)     user_profiles (hub)
├── amazon_review_signals ──┐                              ┌── 10,291 users
├── email_signals ──────────┼── 10,478 evidence ───────────┤
├── shopify_order_signals ──┤                              └── tag_dictionary (215 tags)
└── shopify_review_signals ─┘
amazon_reviewer_history_signals ── Amazon reviewer history background
```

### Key Tables

| Table | Purpose | Rows |
|-------|---------|------|
| `evidence_index` | Unified search layer | 10,478 |
| `user_profiles` | User personas | 10,291 |
| `amazon_reviewer_history_signals` | Amazon reviewer history background | 26,936 |
| `tag_dictionary` | Tag dictionary | 215 |

### 6 Evidence Dimensions (`*_mapped` text[] columns)

| Column | Meaning | Examples |
|--------|---------|----------|
| `pain_points_mapped` | Pain points | battery_life_too_short, insufficient_warmth |
| `strengths_mapped` | Strengths | warmth, overall_satisfaction |
| `use_cases_mapped` | Use cases | outdoor_manual_labor, winter_season_use |
| `purchase_motivations_mapped` | Purchase motivations | gift_for_family, cold_weather_need |
| `occupations_mapped` | Occupations | construction_worker, outdoor_recreation |
| `demographic_signals_mapped` | Demographics | male, senior, female |

### Data Sources

| source_type | Count | Has text |
|-------------|-------|----------|
| amazon_review | 5,245 | Yes |
| email | 211 | Yes (Shopify 159 + Amazon 52) |
| shopify_order | 4,929 | No (transaction data) |
| shopify_review | 93 | Yes |

**Note:** user_profiles uses `std_*` prefix columns; evidence_index uses `*_mapped` suffix columns.

### Amazon Reviewer History

`amazon_reviewer_history_signals` contains historical Amazon reviews from reviewers linked to Dewbu users. Use it as background context for long-term interests, category affinity, brand exposure, and review habits.

Important fields: `user_id`, `history_review_id`, `product_brand`, `product_name`, `star`, `review_time`, `title`, `content`, `direct_review_url`.

`user_profiles.history_review_count > 0` means the user has history rows. Do not require this for all analysis; many useful Dewbu users have no history. Prefer it when the question asks about lifestyle, usual interests, "what else do they buy", or when building a richer interview persona.

History review labels are not standardized across Dewbu dimensions because the product surface is broad. Treat history `pain_points`, `use_cases`, and similar arrays as weak hints; rely mainly on product names, brands, ratings, titles, and review text.

---

## Query Patterns

### Pattern 1: Fuzzy to Precise

```bash
dewbu tags search <keyword>           # Explore available tags
dewbu stats tags --group-by tag       # See distribution
dewbu evidence search --query <kw>    # Broad search
dewbu evidence search --pain-points <specific_tag> --source amazon_review  # Narrow
```

### Pattern 2: Persona Building

```bash
dewbu profile search --occupations hunter --limit 20
dewbu evidence search --occupations hunter --limit 30
dewbu stats tags --group-by tag --source amazon_review --top 20
```

When the user asks about non-Dewbu category interests, enrich with reviewer history:

```bash
dewbu sql "WITH target_users AS (
  SELECT user_id FROM user_profiles
  WHERE history_review_count > 0
    AND EXISTS (SELECT 1 FROM unnest(std_occupations) t WHERE t ILIKE '%hunter%')
)
SELECT h.product_brand, h.product_name, count(*) AS reviews, round(avg(h.star)::numeric, 2) AS avg_star
FROM amazon_reviewer_history_signals h
JOIN target_users u USING(user_id)
GROUP BY h.product_brand, h.product_name
ORDER BY reviews DESC
LIMIT 30"
```

### Pattern 3: Cross-Channel Comparison

```bash
dewbu evidence search --query <kw> --source amazon_review --limit 10
dewbu evidence search --query <kw> --source email --limit 10
```

### Pattern 4: Star Rating Analysis

```bash
dewbu evidence search --query <kw> --star-min 1 --star-max 2 --limit 20  # Low stars
dewbu evidence search --query <kw> --star-min 4 --limit 20               # High stars
```

### Pattern 5: High-Value Users

```bash
dewbu profile search --spend-min 500 --limit 20
dewbu profile search --order-min 3 --limit 20
dewbu profile search --spend-min 200 --pain-points battery --limit 10
```

### Pattern 6: Evidence Retrieval

```bash
dewbu evidence search --query <kw> --limit 5
dewbu evidence get "ev::amazon_review::review::R1234..."
```

### Pattern 7: Custom SQL

```bash
dewbu sql "SELECT brand, unnest(pain_points_mapped) as pain, count(*) FROM evidence_index WHERE brand IS NOT NULL GROUP BY brand, pain ORDER BY count(*) DESC LIMIT 20"
```

---

## Output Format

All commands return JSON by default with `meta` (total, returned, filter) and `data` array.
