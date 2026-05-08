# Dewbu Persona Skill

Agent skill for querying Dewbu consumer insights database. Provides structured access to user personas, evidence (reviews, emails, orders), and tag-based analytics.

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
