#!/usr/bin/env bash
set -euo pipefail

# Dewbu Persona Skill — Installer
# Downloads dewbu binary + skills from the latest GitHub Release.

REPO="reorc/dewbu-persona-skill"
INSTALL_DIR="${DEWBU_INSTALL_DIR:-$HOME/.local/bin}"
SKILLS_DIRS=("$HOME/.claude/skills" "$HOME/.agents/skills")

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "==> Detecting platform: ${OS}_${ARCH}"

# Get latest release tag
echo "==> Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
VERSION="${LATEST#v}"
echo "    Version: ${LATEST}"

# Download and install binary
BINARY_URL="https://github.com/${REPO}/releases/download/${LATEST}/dewbu_${VERSION}_${OS}_${ARCH}.tar.gz"
echo "==> Downloading dewbu binary..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$BINARY_URL" | tar -xz -C "$INSTALL_DIR" dewbu
chmod +x "$INSTALL_DIR/dewbu"
echo "    Installed: $INSTALL_DIR/dewbu"

# Download and install skills
SKILLS_URL="https://github.com/${REPO}/releases/download/${LATEST}/dewbu-skills_${VERSION}.tar.gz"
echo "==> Downloading skills..."
TMP_SKILLS=$(mktemp -d)
curl -fsSL "$SKILLS_URL" | tar -xz -C "$TMP_SKILLS"

for dir in "${SKILLS_DIRS[@]}"; do
  mkdir -p "$dir"
  for skill in "$TMP_SKILLS"/dewbu-*; do
    skill_name=$(basename "$skill")
    rm -rf "$dir/$skill_name"
    cp -r "$skill" "$dir/$skill_name"
    echo "    Installed skill: $dir/$skill_name"
  done
done
rm -rf "$TMP_SKILLS"

echo ""
echo "==> Done! dewbu ${LATEST} installed."
echo "    Binary: $INSTALL_DIR/dewbu"
echo "    Skills: ${SKILLS_DIRS[*]}"
echo ""

# Check PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo "    Note: Add $INSTALL_DIR to your PATH:"
  echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
fi
