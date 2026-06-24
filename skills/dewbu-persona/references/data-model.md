# Dewbu 数据模型速查

## 数据库

`dewbu_persona_v2`，通过 `voc` CLI 查询；默认使用 Dewbu HTTP API，本地配置保存在 `~/.voc/config.json`。

## 架构

```
source tables (spoke)          evidence_index (serving)     user_profiles (hub)
├── amazon_review_signals ──┐                              ┌── 10,291 用户
├── email_signals ──────────┼── 10,478 evidence ───────────┤
├── shopify_order_signals ──┤                              └── tag_dictionary (215 标签)
└── shopify_review_signals ─┘
amazon_reviewer_history_signals ── Amazon reviewer 历史评论背景信号
```

## evidence_index（核心检索表）

| 列 | 类型 | 说明 |
|----|------|------|
| evidence_id | text PK | 唯一标识 |
| source_type | text | amazon_review / email / shopify_review / shopify_order |
| signal_record_id | text | 关联信号表主键 |
| user_id | text | 关联用户（可空） |
| platform | text | Amazon / Shopify |
| brand | text | 品牌 |
| country | text | 国家 |
| event_time | timestamptz | 事件时间 |
| star | integer | 星级（1-5，仅 review） |
| title | text | 标题 |
| content_snippet | text | 内容摘要（前 200 字符） |
| pain_points_mapped | text[] | 映射后痛点 |
| strengths_mapped | text[] | 映射后优势 |
| use_cases_mapped | text[] | 使用场景 |
| purchase_motivations_mapped | text[] | 购买动机 |
| occupations_mapped | text[] | 职业 |
| demographic_signals_mapped | text[] | 人口统计 |
| extraction_confidence | numeric | 提取置信度 |
| content_text | text | 完整原文 |
| content_tsv | tsvector | 全文检索向量（已建 GIN 索引） |

## user_profiles（用户画像表）

| 列 | 类型 | 说明 |
|----|------|------|
| user_id | text PK | 用户唯一标识 |
| source_types | text[] | 来源渠道列表 |
| is_merged_cross_channel | boolean | 是否跨渠道合并 |
| order_count | integer | 订单总数 |
| total_spend | numeric | 累计消费 |
| first_seen_at / last_seen_at | timestamptz | 时间范围 |
| pain_points ~ commercial_value_signals | text[] | 10 个原始标签列 |
| std_pain_points ~ std_commercial_value_signals | text[] | 10 个标准化标签列 |
| countries | text[] | 国家列表 |
| product_names | text[] | 购买产品 |
| inferred_gender | text | 推断性别 |
| inferred_age_range | text | 推断年龄段 |
| history_review_count | integer | 抓到的 Amazon 历史评论数，> 0 表示可联查历史评论 |

**注意**: user_profiles 使用 `std_*` 前缀的列名，evidence_index 使用 `*_mapped` 后缀的列名。

## amazon_reviewer_history_signals（Amazon 历史评论背景表）

这个表记录与 Dewbu 用户关联的 Amazon reviewer 历史评论，不限 Dewbu 产品。它适合补充用户长期兴趣、常买品类、品牌接触、评论习惯和生活方式背景。

| 列 | 类型 | 说明 |
|----|------|------|
| history_signal_id | text | 历史信号 ID |
| history_review_id | text | 历史评论 ID |
| reviewer_id | text | Amazon reviewer ID |
| source_review_id | text | 来源评论 ID |
| source_signal_record_id | text | 来源信号记录 ID |
| user_id | text | 关联 user_profiles |
| asin | text | Amazon ASIN |
| product_brand | text | 商品品牌 |
| product_name | text | 商品名称 |
| star | integer | 星级 |
| reviewed_country | text | 评论国家 |
| review_time | timestamptz | 评论时间 |
| title | text | 评论标题 |
| content | text | 评论正文 |
| direct_review_url | text | 评论链接 |
| verified_purchase | boolean | 是否验证购买 |
| pain_points / strengths / use_cases / purchase_motivations / occupations / demographic_signals | text[] | 历史评论抽取标签，未按 Dewbu 标准化 |

**使用边界**:

- 历史评论是背景信号，不是 Dewbu 产品反馈证据。
- 由于商品面很广，历史标签不标准化；分析时优先使用 `product_brand`、`product_name`、`star`、`title`、`content`。
- 不要默认要求 `history_review_count > 0`，除非用户问题涉及生活方式、平时兴趣、其他品类或需要更立体的访谈 persona。

## tag_dictionary（标签字典）

| 列 | 说明 |
|----|------|
| tag_id | 自增主键 |
| dimension | 维度：pain_points / strengths / use_cases / purchase_motivations / occupations / demographic_signals |
| tag_value | 标签值 |
| evidence_count | evidence 中出现次数 |
| user_count | 用户画像中出现次数 |

## 数据量

| source_type | evidence 数 | 有原文 | 说明 |
|-------------|-------------|--------|------|
| amazon_review | 5,245 | 是 | Amazon 评论，有星级 |
| email | 211 | 是 | 客服邮件（Shopify 159 + Amazon 52） |
| shopify_order | 4,929 | 否 | 纯交易数据，无文本 |
| shopify_review | 93 | 是 | Shopify 评论 |

Amazon 历史评论背景表：`amazon_reviewer_history_signals` 26,936 条；`user_profiles.history_review_count > 0` 的用户 2,008 个。

## 标签维度分布

| 维度 | 标签数 | evidence 提及次数 |
|------|--------|-------------------|
| demographic_signals | 41 | 2,564 |
| pain_points | 40 | 6,187 |
| strengths | 35 | 9,877 |
| use_cases | 35 | 3,255 |
| purchase_motivations | 35 | 1,753 |
| occupations | 29 | 205 |

## 直接 SQL 查询

可以使用 `voc sql` 命令执行任意 SELECT 查询：

```bash
# 品牌分布
voc sql "SELECT brand, count(*) FROM amazon_review_signals GROUP BY brand ORDER BY count(*) DESC"

# 痛点 Top 10
voc sql "SELECT tag_value, evidence_count FROM tag_dictionary WHERE dimension = 'pain_points' ORDER BY evidence_count DESC LIMIT 10"

# 跨表联查
voc sql "SELECT e.source_type, e.title, e.content_snippet, u.total_spend
FROM evidence_index e JOIN user_profiles u USING(user_id)
WHERE u.total_spend > 500 LIMIT 10"

# 猎人群体的 Amazon 历史评论品类/品牌兴趣
voc sql "WITH hunter_users AS (
  SELECT user_id FROM user_profiles
  WHERE history_review_count > 0
    AND EXISTS (SELECT 1 FROM unnest(std_occupations) t WHERE t ILIKE '%hunter%')
)
SELECT h.product_brand, h.product_name, count(*) AS reviews, round(avg(h.star)::numeric, 2) AS avg_star
FROM amazon_reviewer_history_signals h
JOIN hunter_users u USING(user_id)
GROUP BY h.product_brand, h.product_name
ORDER BY reviews DESC
LIMIT 30"
```
