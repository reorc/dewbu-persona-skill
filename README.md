# Dewbu Persona Skill

Agent skill for querying Dewbu consumer insights database. Provides structured access to user personas, evidence (reviews, emails, orders), and tag-based analytics.

## API access

The CLI talks to the Dewbu HTTP API, so CLI users do not need a PostgreSQL connection string.

Create an API key in the Dewbu webapp first:

1. Open `https://dewbu-persona.tool.reorc.cloud/`
2. Go to `Admin` → `Accounts`
3. Open `More` → `API keys` for any **active** account
4. Generate a key and copy it once

**The key inherits the account's permissions:**

| Account role | Can do |
|--------------|--------|
| admin        | everything: SQL queries, list/get personas, **create / update / delete / build personas** |
| user         | read-only: SQL queries, list/get personas |

So if you need an agent to *manage* personas, generate the key from an admin
account; for a read-only analyst agent, use a regular user account.

```bash
dewbu config set \
  --svc-base-url https://dewbu-persona.tool.reorc.cloud/ \
  --api-key dewbu_live_long_random_key

dewbu sql "SELECT count(*) FROM evidence_index"
dewbu evidence search --query battery --limit 5
```

The config file is stored at `~/.dewbu/config.json` by default. You can override it with `DEWBU_CONFIG`.
Environment variables and flags still work and take precedence: `DEWBU_API_BASE_URL` and `DEWBU_API_KEY`.

> Note: there is only one backend now (HTTP). The historical `--backend` flag /
> `DEWBU_BACKEND` env var are accepted but ignored for backward compatibility.

## Persona management

Saved personas can be managed directly from the CLI (admin key required for
writes):

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

`--filter` takes a persona filter config as a JSON object (same shape the webapp
uses): `brands`, `platforms`, `stars`, `countries`, `timeStart`, `timeEnd`,
`tags` (`[{"dimension","values"}]`), `gender`, `ageRange`, `spendRange`,
`orderCountRange`, `sourceTypes`. Changing the filter invalidates the cached
profile, so run `persona build` afterwards to refresh stats. A read-only key
that attempts a write gets a clear `403 — needs an admin API key` message.

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
