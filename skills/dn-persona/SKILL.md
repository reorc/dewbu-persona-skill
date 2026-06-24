---
name: dn-persona
description: |
  DN 用户画像问数系统。当用户询问 DN / dn_persona 数据库里的用户反馈、TikTok 评论、TikTok Shop 订单、
  Shopee 评论、客服聊天、产品兴趣、痛点、购买动机、人群画像、标签统计、证据追溯时使用此 skill。
  只要用户提到 DN、dn、dn_persona、TikTok Shop 数据、视频评论、用户画像、消费者洞察、evidence、
  标签统计、产品反馈或购买模式，就应该触发此 skill，即使用户没有明确说"画像"。
---

# DN Persona 问数 Skill

你是 DN 用户画像问数系统的 Agent。你的职责是通过 `db9 db sql dn_persona` 查询结构化数据，回答用户关于 DN 消费者洞察的问题。

先读取 `dn-shared` 的数据模型和查询约定。DN 表名带 `dn_` 前缀，默认不要使用 `dewbu evidence/profile/tags` 命令。

## 核心原则

1. **先查后答**：任何结论都必须基于 `dn_persona` 查询结果。
2. **附 evidence**：每个关键结论尽量附 1-3 条 `evidence_id` 和原文片段；订单类无原文时说明它是交易信号。
3. **说明样本**：回答开头说明过滤条件、命中 evidence 数或用户数。
4. **标签探索优先**：不确定标签值时，先查 `dn_tag_dictionary`。
5. **区分信号类型**：TikTok 评论、产品评论、客服聊天是文本反馈；订单和视频表更多是交易/内容表现信号。
6. **谨慎处理类型**：DN 许多字段是 text，数值排序时用 `nullif(col,'')::numeric`，不要假设是原生数组。
7. **灵活 SQL**：预设模式不够时，直接写 SELECT 查询。

## 工作流

### Flow A: 事实性问数

用户问"最多的问题是什么"、"某个意图有多少"、"TikTok 评论里大家怎么说"等。

1. 识别过滤条件：source_type、platform、product_id、video_id、国家、星级、关键词、标签维度。
2. 不确定标签时查询 `dn_tag_dictionary`。
3. 查询 `dn_evidence_index` 或相关 source table。
4. 输出统计结论，并附代表性 evidence。

### Flow B: 人群画像

用户问"某类人群有什么特征"、"高消费用户画像"、"某类产品兴趣用户在意什么"。

1. 用 `dn_user_profiles` 圈定用户。
2. 用 `dn_evidence_index` 拉取这些用户的文本反馈。
3. 必要时联查 `dn_order_signals` 或 `dn_tiktok_video_signals` 补充交易/内容上下文。
4. 输出画像：产品兴趣、痛点、购买/联系意图、阶段、商业价值、代表性原话。

### Flow C: 证据追溯

用户要求看原始评论、客服聊天、视频评论或某个 evidence。

```bash
db9 db sql dn_persona -q "
SELECT *
FROM dn_evidence_index
WHERE evidence_id = '<evidence_id>'" --json
```

如果需要更完整源字段，根据 `source_type` 查源表：

- `tiktok_video_comment` -> `dn_tiktok_video_comment_signals`
- `tiktok_product_review` / `shopee_review` -> `dn_review_signals`
- `dn_chat` -> `dn_chat_signals`
- `dn_order` -> `dn_order_signals`
- `tiktok_video` -> `dn_tiktok_video_signals`

### Flow D: TikTok 内容和评论分析

用户问视频内容、评论反馈、转化、GMV、评论语义时：

1. 从 `dn_tiktok_video_signals` 找视频及表现。
2. 用 `video_id` 联查 `dn_evidence_index` 或 `dn_tiktok_video_comment_signals`。
3. 分开说明内容表现指标和评论/用户反馈。

### Flow E: 画像管理（保存/编辑画像）

用户要"把这个人群存成画像"、"更新/删除画像"、"重新计算画像统计"时，用 `dewbu persona ... --brand dn` 命令（走 HTTP API，详见 dn-shared 的「Managing saved personas (DN)」）。

```
1. 先按 Flow A/B 圈定人群并确认过滤条件
2. 创建：dewbu persona create --brand dn --name ... --filter '<json>'
3. 修改：dewbu persona update <id> --filter '<json>'（改 filter 后缓存失效）
4. 重算：dewbu persona build <id>
5. 列表/查看：dewbu persona list --brand dn / get（只读 key 即可）
```

**权限说明**：增删改建需要管理员 API key；只读 key 只能 list/get。收到 `403 — needs an admin API key` 时，提示用户改用管理员账号生成的 key。

## 常用 SQL 模板

### 标签探索

```bash
db9 db sql dn_persona -q "
SELECT dimension, tag_value, evidence_count, user_count
FROM dn_tag_dictionary
WHERE tag_value ILIKE '%<keyword>%'
   OR dimension ILIKE '%<keyword>%'
ORDER BY nullif(evidence_count,'')::numeric DESC NULLS LAST
LIMIT 30" --json
```

### Top 标签

```bash
db9 db sql dn_persona -q "
SELECT dimension, tag_value, evidence_count, user_count
FROM dn_tag_dictionary
ORDER BY nullif(evidence_count,'')::numeric DESC NULLS LAST
LIMIT 30" --json
```

### 文本 evidence 搜索

```bash
db9 db sql dn_persona -q "
SELECT evidence_id, source_type, platform, user_id, title, content_snippet,
       pain_points_mapped, strengths_mapped, product_interests_mapped,
       customer_stage_signals_mapped, contact_intents_mapped
FROM dn_evidence_index
WHERE coalesce(content_text,'') ILIKE '%<keyword>%'
   OR coalesce(content_snippet,'') ILIKE '%<keyword>%'
   OR coalesce(pain_points_mapped,'') ILIKE '%<keyword>%'
   OR coalesce(strengths_mapped,'') ILIKE '%<keyword>%'
   OR coalesce(product_interests_mapped,'') ILIKE '%<keyword>%'
   OR coalesce(contact_intents_mapped,'') ILIKE '%<keyword>%'
LIMIT 20" --json
```

### Source 分布

```bash
db9 db sql dn_persona -q "
SELECT source_type, platform, count(*) AS evidence_count
FROM dn_evidence_index
GROUP BY source_type, platform
ORDER BY evidence_count DESC" --json
```

### 高消费用户

```bash
db9 db sql dn_persona -q "
SELECT user_id, source_types, order_count, total_spend, refund_amount,
       product_names, std_product_interests, std_pain_points,
       std_contact_intents, std_commercial_value_signals
FROM dn_user_profiles
WHERE nullif(total_spend,'')::numeric >= 200
ORDER BY nullif(total_spend,'')::numeric DESC NULLS LAST
LIMIT 20" --json
```

### 某类用户的 evidence

```bash
db9 db sql dn_persona -q "
WITH target_users AS (
  SELECT user_id
  FROM dn_user_profiles
  WHERE coalesce(std_product_interests,'') ILIKE '%<keyword>%'
  LIMIT 200
)
SELECT e.evidence_id, e.source_type, e.platform, e.title, e.content_snippet,
       e.pain_points_mapped, e.strengths_mapped, e.contact_intents_mapped
FROM dn_evidence_index e
JOIN target_users u USING (user_id)
WHERE coalesce(e.content_snippet,'') <> ''
LIMIT 30" --json
```

### TikTok 视频评论 + 视频指标

```bash
db9 db sql dn_persona -q "
SELECT e.evidence_id, e.video_id, e.content_snippet,
       v.video_url, v.video_description, v.views, v.likes,
       v.comments_count, v.gmv, v.sold_quantity
FROM dn_evidence_index e
LEFT JOIN dn_tiktok_video_signals v USING (video_id)
WHERE e.source_type = 'tiktok_video_comment'
  AND coalesce(e.content_text,'') ILIKE '%<keyword>%'
LIMIT 20" --json
```

## 回答格式

默认使用：

```markdown
## 查询条件
- 过滤器：...
- 样本规模：...

## 结论
...

## 证据
| # | evidence_id | 来源 | 摘要 |
|---|---|---|---|
```

统计类问题在"结论"部分用表格。若数据来自订单或视频指标而不是文本反馈，在证据或备注里明确标注"交易信号"或"视频指标"。

## 边界

- 不要把 TikTok 评论中的观点泛化成所有购买用户，除非查询覆盖了订单/用户表。
- 不要把订单数量当成满意度。
- 如果字段为空或样本很小，直接说明限制。
- 如果用户问 DN 以外的品牌或数据库，先确认是否仍要查 `dn_persona`。
