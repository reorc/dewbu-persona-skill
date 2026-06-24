# VOC Persona Skill

Agent skill bundle for querying brand-specific consumer insight deployments. Provides structured access to user personas, evidence, and tag-based analytics for Dewbu, DN, and future brands.

## API access

The `voc` CLI talks to the configured HTTP API, so CLI users do not need a PostgreSQL connection string.

Create an API key in the target brand webapp first:

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
voc config set \
  --svc-base-url https://dewbu-persona.tool.reorc.cloud/ \
  --api-key dewbu_live_long_random_key

voc sql "SELECT count(*) FROM evidence_index"
voc evidence search --query battery --limit 5
```

The config file is stored at `~/.voc/config.json` by default. You can override it with `VOC_CONFIG`.
Environment variables and flags still work and take precedence: `VOC_API_BASE_URL` and `VOC_API_KEY`. Legacy `DEWBU_*` environment variables are still accepted for compatibility.

> Note: there is only one backend now (HTTP). The historical `--backend` flag and backend env vars are accepted but ignored for backward compatibility.

## Persona management

Saved personas can be managed directly from the CLI (admin key required for
writes):

```bash
# read-only (any key)
voc persona list
voc persona get <persona_id>

# writes (admin key only)
voc persona create --name "Hunters" \
  --description "Cold-weather hunting users" \
  --filter '{"occupations":["hunter"],"stars":[1,2]}'
voc persona update <persona_id> --name "Hunters v2" --filter '{"stars":[1,2,3]}'
voc persona build  <persona_id>   # recompute cached profile/stats
voc persona delete <persona_id>
```

`--filter` takes a persona filter config as a JSON object (same shape the webapp
uses): `brands`, `platforms`, `stars`, `countries`, `timeStart`, `timeEnd`,
`tags` (`[{"dimension","values"}]`), `gender`, `ageRange`, `spendRange`,
`orderCountRange`, `sourceTypes`. Changing the filter invalidates the cached
profile, so run `voc persona build` afterwards to refresh stats. A read-only key
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
