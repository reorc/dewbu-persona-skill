# VOC CLI 命令参考

## 全局参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--format` | `json` | 输出格式：json / table / csv |
| `--limit` | `20` | 最大返回行数 |
| `--offset` | `0` | 分页偏移 |
| `--fields` | 全部 | 逗号分隔的返回字段 |
| `--svc-base-url` | config/env | API 服务地址，例如 `https://<deployment-domain>/` |
| `--api-key` | config/env | API key |

> 连哪个品牌完全由配置的 `svc_base_url` + `api_key` 决定（每个部署单品牌、独立域名），命令里不需要任何品牌/数据库选择参数。

配置保存在 `~/.voc/config.json`。先在对应部署的 webapp `Admin` -> `Accounts` -> `More` -> `API keys` 里生成 API key（管理员账号可管理画像，普通用户为只读），再写入 CLI：

```bash
voc config set --svc-base-url https://<deployment-domain>/ --api-key dewbu_live_xxx
voc config show
```

---

## voc sql \<query\>

执行任意只读 SQL 查询（SELECT / WITH ... SELECT）。写操作会被拒绝。

```bash
voc sql "SELECT count(*) FROM evidence_index"
voc sql "SELECT brand, count(*) FROM amazon_review_signals GROUP BY brand ORDER BY count(*) DESC"
voc sql "SELECT tag_value, evidence_count FROM tag_dictionary WHERE dimension = 'pain_points' ORDER BY evidence_count DESC LIMIT 10"
voc sql "WITH top_users AS (SELECT user_id, total_spend FROM user_profiles ORDER BY total_spend DESC LIMIT 5) SELECT * FROM top_users"
```

当预定义命令无法满足查询需求时，优先使用 `voc sql`。

---

## voc tags search \<keyword\>

跨 tag_dictionary 和所有 `*_mapped` 列做子串搜索，帮助发现可用标签值。

```bash
voc tags search battery
voc tags search gift
voc tags search cold
```

返回两部分：
- `dictionary`：tag_dictionary 中匹配的标签（含 dimension + evidence_count + user_count）
- `usage`：evidence_index 各 `*_mapped` 列中实际使用的值 + 出现次数

---

## voc evidence search

### 搜索模式

| Flag | 搜索方式 | 适用场景 |
|------|----------|----------|
| `--query <keyword>` | 跨维度 ILIKE + FTS 分层搜索 | 不确定标签在哪个维度 |
| `--pain-points <keyword>` | 只搜 pain_points_mapped（ILIKE） | 明确要搜痛点 |
| `--strengths <keyword>` | 只搜 strengths_mapped（ILIKE） | 明确要搜优势 |
| `--use-cases <keyword>` | 只搜 use_cases_mapped | 明确要搜使用场景 |
| `--purchase-motivations <keyword>` | 只搜 purchase_motivations_mapped | 搜购买动机 |
| `--occupations <keyword>` | 只搜 occupations_mapped | 搜职业 |
| `--demographic <keyword>` | 只搜 demographic_signals_mapped | 搜人口统计 |

### 结构化过滤

| Flag | 说明 |
|------|------|
| `--source` | 来源：amazon_review / email / shopify_order / shopify_review |
| `--star-min` / `--star-max` | 星级范围（1-5） |
| `--country` | 国家 |
| `--user` | 用户 ID 精确匹配 |

### JSON Filter DSL

```bash
voc evidence search --filter '{
  "source_type": "amazon_review",
  "star": {"gte": 4},
  "brand": "DEWBU",
  "time": {"after": "2024-01-01"}
}'
```

### 输出字段

`match_tier`：1 = 标签命中（更精准），2 = FTS 原文命中
`fts_rank`：FTS 相关性分数（tier 2 时有值）

---

## voc evidence get \<evidence_id\>

获取单条 evidence 完整信息，含原文。

```bash
voc evidence get "evidence::amazon_review::review::https://www.amazon.com/gp/customer-reviews/R1IF9O7VQVGA6G"
```

---

## voc profile search

### 搜索模式

与 evidence search 类似的维度 flag（`--query`, `--pain-points`, `--strengths` 等）。
注意：profile 搜索的维度 flag 在 user_profiles 表的 `std_*` 列上做 ILIKE。

### 额外过滤

| Flag | 说明 |
|------|------|
| `--source` | 过滤 source_types 数组 |
| `--spend-min` / `--spend-max` | 消费金额范围 |
| `--order-min` | 最低订单数 |

### 示例

```bash
voc profile search --query battery --limit 10
voc profile search --spend-min 200 --pain-points battery
voc profile search --occupations hunter --limit 20
```

---

## voc profile schema

返回可用过滤字段和标签枚举（从 tag_dictionary 动态统计）。

```bash
voc profile schema
voc profile schema --dimension pain_points
voc profile schema --dimension strengths
```

---

## voc stats tags

标签分布统计。

```bash
voc stats tags --group-by dimension
voc stats tags --group-by tag --top 20
```

| 参数 | 说明 |
|------|------|
| `--group-by` | dimension（默认）或 tag |
| `--top` | 返回 top N（默认 30） |
