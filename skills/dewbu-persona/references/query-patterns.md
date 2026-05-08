# 常见查询模式

## 模式 1：从模糊到精确

用户问题往往模糊，需要逐步收窄：

```bash
# Step 1: 探索有什么相关标签
dewbu tags search <用户提到的关键词>

# Step 2: 看整体分布
dewbu stats tags --group-by tag --top 20

# Step 3: 用 --query 做宽泛搜索
dewbu evidence search --query <keyword> --limit 20

# Step 4: 用维度 flag 精确搜索
dewbu evidence search --pain-points <specific_tag> --source amazon_review
```

## 模式 2：人群画像构建

```bash
# Step 1: 找到目标人群
dewbu profile search --occupations hunter --limit 20

# Step 2: 看这个人群的 evidence
dewbu evidence search --occupations hunter --limit 30

# Step 3: 看痛点分布
dewbu evidence search --occupations hunter --pain-points "" --limit 50
# 或者用 stats
dewbu stats tags --group-by tag --source amazon_review --top 20
```

## 模式 3：跨渠道对比

```bash
# Amazon 评论中的反馈
dewbu evidence search --query <keyword> --source amazon_review --limit 10

# 邮件中的反馈
dewbu evidence search --query <keyword> --source email --limit 10

# 或者用 SQL 做更灵活的统计
dewbu sql "SELECT source_type, count(*) FROM evidence_index WHERE EXISTS (SELECT 1 FROM unnest(pain_points_mapped) t WHERE t ILIKE '%battery%') GROUP BY source_type"
```

## 模式 4：星级对比分析

```bash
# 低星评论的痛点
dewbu evidence search --query <keyword> --star-min 1 --star-max 2 --limit 20

# 高星评论的优势
dewbu evidence search --query <keyword> --star-min 4 --limit 20
```

## 模式 5：高价值用户分析

```bash
# 高消费用户画像
dewbu profile search --spend-min 500 --limit 20

# 复购用户
dewbu profile search --order-min 3 --limit 20

# 高消费 + 有痛点的用户（流失风险）
dewbu profile search --spend-min 200 --pain-points battery --limit 10
```

## 模式 6：证据追溯

当需要看原始评论/邮件时：

```bash
# 先搜索找到相关 evidence
dewbu evidence search --query <keyword> --limit 5

# 从结果中取 evidence_id，获取完整原文
dewbu evidence get "ev::amazon_review::review::R1234..."
```

## 查询组合规则

- 所有 flag 可以自由组合（AND 关系）
- `--query` 与维度 flag 可以同时使用（先按维度过滤，再在结果中搜索）
- `--tag` 是精确匹配，`--pain-points` 等是 ILIKE 模糊匹配
- `--limit` 和 `--offset` 用于分页
- 结果中 `meta.total` 告诉你总共有多少条命中

## 模式 7：自定义 SQL 查询

当预定义命令不够灵活时，用 `dewbu sql` 执行任意 SELECT：

```bash
# 品牌 × 痛点交叉统计
dewbu sql "SELECT brand, unnest(pain_points_mapped) as pain, count(*) FROM evidence_index WHERE brand IS NOT NULL GROUP BY brand, pain ORDER BY count(*) DESC LIMIT 20"

# 高消费用户的评论
dewbu sql "SELECT e.title, e.content_snippet, u.total_spend FROM evidence_index e JOIN user_profiles u USING(user_id) WHERE u.total_spend > 500 AND e.source_type = 'amazon_review' ORDER BY u.total_spend DESC LIMIT 10"

# 邮件平台分布
dewbu sql "SELECT platform, count(*) FROM email_signals GROUP BY platform"
```
