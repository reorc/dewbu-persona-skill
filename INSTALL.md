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

## Step 4: Install the Skill

### Method 1: Git Clone (Recommended)

```bash
# For Claude Code
git clone https://github.com/reorc/dewbu-persona-skill.git ~/.claude/skills/dewbu-persona

# For OpenClaw / OpenCode / other Agents
git clone https://github.com/reorc/dewbu-persona-skill.git ~/.agents/skills/dewbu-persona
```

### Method 2: Download ZIP

```bash
curl -L https://github.com/reorc/dewbu-persona-skill/archive/refs/heads/main.zip -o /tmp/dewbu-persona.zip
unzip /tmp/dewbu-persona.zip -d ~/.claude/skills/
mv ~/.claude/skills/dewbu-persona-skill-main ~/.claude/skills/dewbu-persona
```

---

## Updating

```bash
cd ~/.claude/skills/dewbu-persona  # or ~/.agents/skills/dewbu-persona
git pull
```

---

## Verification

```bash
dewbu tags search battery
```

Should return JSON with tag matches from the database.

---

## Uninstallation

```bash
# Remove skill
rm -rf ~/.claude/skills/dewbu-persona

# Remove CLI
rm ~/.local/bin/dewbu

# Logout db9 (optional)
db9 logout
```
