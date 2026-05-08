# Dewbu Persona Skill — Installation Guide

## Step 1: Install db9

Read https://db9.ai/skill.md and follow the instructions to install db9 CLI.

Quick install:

```bash
curl -fsSL https://db9.ai/install.sh | sh
```

Verify:

```bash
db9 --version
```

---

## Step 2: Install dewbu CLI

Download the latest `dewbu` binary from GitHub releases and install to `~/.local/bin`:

```bash
# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

# Download latest release
curl -L "https://github.com/reorc/dewbu-persona-skill/releases/latest/download/dewbu-${OS}-${ARCH}" -o /tmp/dewbu

# Install
mkdir -p ~/.local/bin
install -m 755 /tmp/dewbu ~/.local/bin/dewbu
rm /tmp/dewbu
```

Ensure `~/.local/bin` is in PATH:

```bash
echo $PATH | grep -q "$HOME/.local/bin" || echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

Verify:

```bash
dewbu --help
```

---

## Step 3: Authenticate with db9

You need a db9 API key to access the dewbu_persona_v2 database.

Ask the user for their API key, then run:

```bash
db9 login --api-key <API_KEY>
```

Verify:

```bash
db9 status
```

You should see the account info and `dewbu_persona_v2` listed.

---

## Step 4: Install the Skills

本项目包含多个独立 skill（`dewbu-persona`、`dewbu-interview`、`dewbu-shared`），需要分别安装到 skills 目录，使每个 skill 都能被 Agent 工具独立识别。

### Method 1: Git Clone + Symlink (Recommended)

先 clone 到本地任意位置，再为每个 skill 创建 symlink：

```bash
# Clone repo to a source location
git clone https://github.com/reorc/dewbu-persona-skill.git ~/.local/share/dewbu-persona-skill

# For Claude Code
mkdir -p ~/.claude/skills
ln -sf ~/.local/share/dewbu-persona-skill/dewbu-persona ~/.claude/skills/dewbu-persona
ln -sf ~/.local/share/dewbu-persona-skill/dewbu-interview ~/.claude/skills/dewbu-interview
ln -sf ~/.local/share/dewbu-persona-skill/dewbu-shared ~/.claude/skills/dewbu-shared

# For OpenClaw / OpenCode / other Agents
mkdir -p ~/.agents/skills
ln -sf ~/.local/share/dewbu-persona-skill/dewbu-persona ~/.agents/skills/dewbu-persona
ln -sf ~/.local/share/dewbu-persona-skill/dewbu-interview ~/.agents/skills/dewbu-interview
ln -sf ~/.local/share/dewbu-persona-skill/dewbu-shared ~/.agents/skills/dewbu-shared
```

### Method 2: One-liner Install Script

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/reorc/dewbu-persona-skill/main/install.sh)
```

### Method 3: Download ZIP

```bash
curl -L https://github.com/reorc/dewbu-persona-skill/archive/refs/heads/main.zip -o /tmp/dewbu-persona.zip
unzip /tmp/dewbu-persona.zip -d /tmp/

# Copy each skill individually
mkdir -p ~/.claude/skills ~/.agents/skills
for skill in dewbu-persona dewbu-interview dewbu-shared; do
  cp -r /tmp/dewbu-persona-skill-main/$skill ~/.claude/skills/$skill
  cp -r /tmp/dewbu-persona-skill-main/$skill ~/.agents/skills/$skill
done

rm -rf /tmp/dewbu-persona-skill-main /tmp/dewbu-persona.zip
```

---

## Updating

```bash
cd ~/.local/share/dewbu-persona-skill
git pull
# Symlinks automatically point to updated content — no extra steps needed
```

---

## Verification

```bash
# Check skill structure
ls ~/.claude/skills/dewbu-*/SKILL.md
ls ~/.agents/skills/dewbu-*/SKILL.md

# Test CLI
dewbu tags search battery
```

Should see `SKILL.md` for each skill, and JSON tag matches from the database.

---

## Uninstallation

```bash
# Remove skill symlinks/copies
rm -rf ~/.claude/skills/dewbu-persona ~/.claude/skills/dewbu-interview ~/.claude/skills/dewbu-shared
rm -rf ~/.agents/skills/dewbu-persona ~/.agents/skills/dewbu-interview ~/.agents/skills/dewbu-shared

# Remove source repo
rm -rf ~/.local/share/dewbu-persona-skill

# Remove CLI
rm ~/.local/bin/dewbu

# Logout db9 (optional)
db9 logout
```
