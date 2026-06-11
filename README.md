# Dewbu Persona Skill

Agent skill for querying Dewbu consumer insights database. Provides structured access to user personas, evidence (reviews, emails, orders), and tag-based analytics.

## Query Backend

The CLI uses the Dewbu HTTP SQL gateway by default, so CLI users do not need a PostgreSQL connection string.

Create an API key in the Dewbu webapp first:

1. Open `https://dewbu-persona.tool.reorc.cloud/`
2. Go to `Admin` → `Accounts`
3. Open `More` → `API keys` for an active admin account
4. Generate a key and copy it once

```bash
dewbu config set \
  --backend http \
  --svc-base-url https://dewbu-persona.tool.reorc.cloud/ \
  --api-key dewbu_live_long_random_key

dewbu sql "SELECT count(*) FROM evidence_index"
dewbu evidence search --query battery --limit 5
```

The config file is stored at `~/.dewbu/config.json` by default. You can override it with `DEWBU_CONFIG`.
Environment variables and flags still work and take precedence: `DEWBU_BACKEND`, `DEWBU_API_BASE_URL`, and `DEWBU_API_KEY`.

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
