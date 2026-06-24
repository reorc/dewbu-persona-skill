---
name: dn-shared
version: 1.0.0
description: "Shared DN Persona guidance for db9 SQL usage, DN data model, query patterns, and conventions. Read this before using dn-persona or dn-interview."
metadata:
  requires:
    bins: ["db9"]
  database: "dn_persona"
---

# DN Shared Guide

Read this before using any DN skill. DN uses the `dn_persona` db9 database and has DN-prefixed tables, so prefer direct `db9 db sql dn_persona` queries instead of the existing `dewbu` CLI commands.

## Command Pattern

Use read-only SQL:

```bash
db9 db sql dn_persona -q "SELECT count(*) FROM dn_evidence_index" --json
db9 db sql dn_persona -q "SELECT source_type, count(*) FROM dn_evidence_index GROUP BY source_type ORDER BY count(*) DESC" --json
db9 db sql dn_persona -q "SELECT dimension, tag_value, evidence_count FROM dn_tag_dictionary WHERE dimension = 'pain_points' ORDER BY nullif(evidence_count,'')::numeric DESC LIMIT 10" --json
```

Do not run write operations unless the user explicitly asks for data modification. Most persona and interview work should only query.

## Managing saved personas (DN)

Saved DN personas are managed through the `dewbu` CLI over HTTP with `--brand dn`
(this is separate from the `db9` query path above). **Permissions follow your
API key**: a read-only (user) key can `list` / `get`; create / update / delete /
build require an **admin** key, and a read-only key attempting a write returns
`403 — needs an admin API key`.

```bash
# read-only (any key)
dewbu persona list --brand dn
dewbu persona get <persona_id>

# writes (admin key only)
dewbu persona create --brand dn --name "..." --filter '{"stars":[1,2]}'
dewbu persona update <persona_id> --filter '{"stars":[1,2,3]}'
dewbu persona build  <persona_id>   # recompute cached profile/stats
dewbu persona delete <persona_id>
```

The CLI must be configured with an API key first (`dewbu config set
--svc-base-url ... --api-key ...`). Generate the key in the webapp under
`Admin → Accounts → API keys`; pick an admin account for management, a regular
user account for read-only access.

## Database

`dn_persona` (db9)

```
source tables (spoke)                  dn_evidence_index (serving)     dn_user_profiles (hub)
├── dn_order_signals ────────────────┐                                   ┌── 43,356 users
├── dn_review_signals ───────────────┤── 44,585 evidence ────────────────┤
├── dn_chat_signals ─────────────────┤                                   └── dn_tag_dictionary (213 tags)
├── dn_tiktok_video_comment_signals ─┤
└── dn_tiktok_video_signals ─────────┘
```

## Key Tables

| Table | Purpose | Rows |
|---|---:|---:|
| `dn_evidence_index` | Unified evidence/search layer | 44,585 |
| `dn_user_profiles` | Merged user profiles | 43,356 |
| `dn_tag_dictionary` | Standard tag dictionary | 213 |
| `dn_order_signals` | TikTok Shop order-derived signals | 31,568 |
| `dn_tiktok_video_comment_signals` | TikTok video comment signals | 12,126 |
| `dn_review_signals` | Product review signals | 422 |
| `dn_chat_signals` | Customer service chat signals | 169 |
| `dn_tiktok_video_signals` | TikTok video performance/content records | 300 |

## Evidence Sources

| source_type | platform | evidence rows | Has text |
|---|---|---:|---|
| `dn_order` | TikTok Shop | 31,568 | No, transaction/product data |
| `tiktok_video_comment` | TikTok | 12,126 | Yes |
| `tiktok_product_review` | TikTok Shop | 408 | Yes |
| `tiktok_video` | TikTok | 300 | Yes, video description/performance metadata |
| `dn_chat` | Customer Service | 169 | Yes |
| `shopee_review` | Shopee | 14 | Yes |

## Core Columns

### `dn_evidence_index`

Important fields:

| Column | Meaning |
|---|---|
| `evidence_id` | Unique evidence ID |
| `source_type` | Source channel, such as `dn_order`, `tiktok_video_comment`, `dn_chat` |
| `signal_record_id` | Source signal table record ID |
| `user_id` | Linked `dn_user_profiles.user_id`, if available |
| `platform`, `brand`, `country` | Source context |
| `event_time`, `star`, `title` | Stored as text in this database; cast carefully when needed |
| `content_snippet`, `content_text` | Text excerpt and full text where available |
| `video_id`, `product_id` | TikTok/video/product identifiers where available |
| `extraction_confidence` | Stored as text; cast only after checking values |

Standard mapped dimensions on evidence:

| Column | Meaning |
|---|---|
| `pain_points_mapped` | Pain points |
| `strengths_mapped` | Strengths |
| `use_cases_mapped` | Use cases |
| `purchase_motivations_mapped` | Purchase motivations |
| `demographic_signals_mapped` | Demographic signals |
| `product_interests_mapped` | Product/category interests |
| `customer_stage_signals_mapped` | Customer journey stage |
| `contact_intents_mapped` | Service/contact intent |
| `commercial_value_signals_mapped` | Commercial value signals |

### `dn_user_profiles`

Important fields:

| Column | Meaning |
|---|---|
| `user_id`, `merge_key` | Profile identifiers |
| `brand`, `source_types`, `source_user_keys` | Merged source context |
| `first_seen_at`, `last_seen_at` | Stored as text |
| `order_count`, `total_spend`, `refund_amount` | Stored as text; cast with `nullif(col,'')::numeric` when needed |
| `countries`, `product_ids`, `product_names`, `variants`, `video_ids` | Product and source context |
| `source_review_count`, `source_chat_count`, `source_video_comment_count`, `source_order_count` | Source contribution counts, stored as text |
| `std_*` columns | Standardized profile dimensions matching the evidence mapped dimensions |

DN stores many list-like columns as text rather than native `text[]`. Inspect values before using array operators.

## Tag Dimensions

| Dimension | Tags | Evidence mentions | User mentions |
|---|---:|---:|---:|
| `commercial_value_signals` | 25 | 13,333 | 13,203 |
| `customer_stage_signals` | 13 | 10,854 | 10,304 |
| `contact_intents` | 25 | 10,158 | 9,683 |
| `demographic_signals` | 25 | 5,720 | 5,404 |
| `product_interests` | 25 | 4,444 | 3,844 |
| `strengths` | 25 | 2,018 | 1,823 |
| `use_cases` | 25 | 1,467 | 1,257 |
| `pain_points` | 25 | 928 | 893 |
| `purchase_motivations` | 25 | 625 | 569 |

## Query Patterns

### Explore Tags First

When the user's wording does not match a known tag, search the dictionary:

```bash
db9 db sql dn_persona -q "
SELECT dimension, tag_value, evidence_count, user_count
FROM dn_tag_dictionary
WHERE tag_value ILIKE '%refund%' OR dimension ILIKE '%refund%'
ORDER BY nullif(evidence_count,'')::numeric DESC NULLS LAST
LIMIT 30" --json
```

### Broad Evidence Search

Search text and mapped dimensions with `ILIKE`. Because DN mapped fields are text, use substring matching instead of `unnest`.

```bash
db9 db sql dn_persona -q "
SELECT evidence_id, source_type, platform, title, content_snippet,
       pain_points_mapped, strengths_mapped, product_interests_mapped
FROM dn_evidence_index
WHERE coalesce(content_text,'') ILIKE '%battery%'
   OR coalesce(content_snippet,'') ILIKE '%battery%'
   OR coalesce(pain_points_mapped,'') ILIKE '%battery%'
   OR coalesce(product_interests_mapped,'') ILIKE '%battery%'
LIMIT 20" --json
```

### Dimension Distribution

For top tags, use `dn_tag_dictionary` first. For a filtered subset, aggregate with text matching or inspect source rows.

```bash
db9 db sql dn_persona -q "
SELECT tag_value, evidence_count, user_count
FROM dn_tag_dictionary
WHERE dimension = 'contact_intents'
ORDER BY nullif(evidence_count,'')::numeric DESC NULLS LAST
LIMIT 20" --json
```

### Persona Building

Use `dn_user_profiles` to find user-level patterns, then pull evidence for concrete quotes.

```bash
db9 db sql dn_persona -q "
SELECT user_id, source_types, order_count, total_spend, product_names,
       std_pain_points, std_strengths, std_product_interests, std_contact_intents
FROM dn_user_profiles
WHERE coalesce(std_product_interests,'') ILIKE '%lash%'
ORDER BY nullif(total_spend,'')::numeric DESC NULLS LAST
LIMIT 20" --json
```

Then retrieve examples:

```bash
db9 db sql dn_persona -q "
SELECT evidence_id, source_type, platform, title, content_snippet
FROM dn_evidence_index
WHERE user_id = '<user_id>'
ORDER BY event_time DESC NULLS LAST
LIMIT 20" --json
```

### TikTok Video Context

When the user asks about content, comments, video performance, or creators, join evidence/comment rows to `dn_tiktok_video_signals` by `video_id`.

```bash
db9 db sql dn_persona -q "
SELECT c.evidence_id, c.content_snippet, v.video_url, v.video_description,
       v.views, v.likes, v.comments_count, v.gmv, v.sold_quantity
FROM dn_evidence_index c
LEFT JOIN dn_tiktok_video_signals v USING (video_id)
WHERE c.source_type = 'tiktok_video_comment'
  AND coalesce(c.content_text,'') ILIKE '%price%'
LIMIT 20" --json
```

## Output Conventions

For any user-facing answer:

- State the query filters and sample size.
- Separate transactional signals from text feedback.
- Support each interpretation with evidence IDs and snippets when text evidence exists.
- Say when a result is based on tags, raw text, or order/video metadata.
- Do not invent demographics, motives, or product claims not present in the queried data.
