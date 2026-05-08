#!/usr/bin/env bash
set -euo pipefail

# Dewbu Persona Skill — Installer
# Clones the repo and symlinks each skill into ~/.claude/skills and ~/.agents/skills

REPO_URL="https://github.com/reorc/dewbu-persona-skill.git"
SOURCE_DIR="$HOME/.local/share/dewbu-persona-skill"
SKILLS=(dewbu-persona dewbu-interview dewbu-shared)

echo "==> Installing dewbu skills..."

# Clone or update source
if [ -d "$SOURCE_DIR/.git" ]; then
  echo "    Source exists, pulling latest..."
  git -C "$SOURCE_DIR" pull --ff-only
else
  echo "    Cloning repo..."
  rm -rf "$SOURCE_DIR"
  git clone "$REPO_URL" "$SOURCE_DIR"
fi

# Create skill directories
mkdir -p "$HOME/.claude/skills" "$HOME/.agents/skills"

# Symlink each skill
for skill in "${SKILLS[@]}"; do
  if [ -d "$SOURCE_DIR/$skill" ]; then
    ln -sfn "$SOURCE_DIR/$skill" "$HOME/.claude/skills/$skill"
    ln -sfn "$SOURCE_DIR/$skill" "$HOME/.agents/skills/$skill"
    echo "    Linked: $skill"
  else
    echo "    Warning: $skill not found in repo, skipping"
  fi
done

echo ""
echo "==> Done! Skills installed:"
ls -1 "$HOME/.claude/skills/" | grep "^dewbu-" | sed 's/^/    /'
echo ""
echo "To update later: cd $SOURCE_DIR && git pull"
