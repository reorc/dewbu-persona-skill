# Dewbu Persona Skill — Installation Guide

## Prerequisites

- [db9 CLI](https://db9.ai/skill.md) installed and authenticated
- A db9 API key with access to `dewbu_persona_v2`

### Install db9

```bash
curl -fsSL https://db9.ai/install.sh | sh
db9 login --api-key <YOUR_API_KEY>
db9 status  # should show dewbu_persona_v2
```

---

## Install (One-liner)

Downloads the latest `dewbu` binary and all skills from GitHub Releases:

```bash
curl -fsSL https://raw.githubusercontent.com/reorc/dewbu-persona-skill/main/install.sh | bash
```

This installs:
- `dewbu` binary → `~/.local/bin/dewbu`
- Skills → `~/.claude/skills/dewbu-*` and `~/.agents/skills/dewbu-*`

Ensure `~/.local/bin` is in your PATH:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

---

## Manual Install

### Binary

Download from [Releases](https://github.com/reorc/dewbu-persona-skill/releases/latest):

```bash
# Example for macOS ARM
curl -fsSL https://github.com/reorc/dewbu-persona-skill/releases/latest/download/dewbu_<VERSION>_darwin_arm64.tar.gz | tar -xz -C ~/.local/bin
```

### Skills

```bash
# Download skills tarball
curl -fsSL https://github.com/reorc/dewbu-persona-skill/releases/latest/download/dewbu-skills_<VERSION>.tar.gz -o /tmp/skills.tar.gz

# Extract to skill directories
mkdir -p ~/.claude/skills ~/.agents/skills
tar -xzf /tmp/skills.tar.gz -C ~/.claude/skills
tar -xzf /tmp/skills.tar.gz -C ~/.agents/skills
rm /tmp/skills.tar.gz
```

---

## Verification

```bash
dewbu version
dewbu tags search battery
ls ~/.claude/skills/dewbu-*/SKILL.md
```

---

## Updating

Re-run the install script — it always fetches the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/reorc/dewbu-persona-skill/main/install.sh | bash
```

---

## Uninstallation

```bash
# Remove binary
rm ~/.local/bin/dewbu

# Remove skills
rm -rf ~/.claude/skills/dewbu-persona ~/.claude/skills/dewbu-interview ~/.claude/skills/dewbu-shared
rm -rf ~/.agents/skills/dewbu-persona ~/.agents/skills/dewbu-interview ~/.agents/skills/dewbu-shared

# Logout db9 (optional)
db9 logout
```
