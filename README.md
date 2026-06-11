# Dewbu Persona Skill

Agent skill for querying Dewbu consumer insights database. Provides structured access to user personas, evidence (reviews, emails, orders), and tag-based analytics.

## Query Backend

The CLI defaults to the original db9 backend. It can also query a Dewbu HTTP SQL gateway, which is useful when the data has moved to Aliyun PostgreSQL and CLI users should not receive a PostgreSQL connection string.

```bash
# Original behavior
dewbu --backend db9 sql "SELECT count(*) FROM evidence_index"

# HTTP gateway behavior
export DEWBU_BACKEND=http
export DEWBU_API_BASE_URL=https://your-api.example.com
export DEWBU_API_KEY=dewbu_live_long_random_key

dewbu sql "SELECT count(*) FROM evidence_index"
dewbu evidence search --query battery --limit 5
```

Prefer environment variables for `DEWBU_API_KEY` instead of the `--api-key` flag so secrets do not land in shell history.

## Skills

| Skill | Description |
|-------|-------------|
| `dewbu-shared` | Shared CLI reference, data model, and query patterns |
| `dewbu-persona` | Consumer insights queries — pain points, personas, evidence |
| `dewbu-interview` | Simulated user interviews based on real consumer data |

## Quick Start

See [INSTALL.md](INSTALL.md) for setup instructions.

## Structure

```
dewbu-persona-skill/
├── INSTALL.md
├── README.md
├── skills/
│   ├── dewbu-shared/SKILL.md    ← Shared: CLI, data model, patterns
│   ├── dewbu-persona/SKILL.md   ← Query skill (requires dewbu-shared)
│   └── dewbu-interview/SKILL.md ← Interview skill (requires dewbu-shared)
└── evals/
    ├── dewbu-persona-evals.json
    └── dewbu-interview-evals.json
```

## License

MIT
