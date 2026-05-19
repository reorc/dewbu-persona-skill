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

# SHA256 helper (macOS uses shasum, Linux uses sha256sum)
sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

# Download checksums and archives to temp dir
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

BASE_URL="https://github.com/${REPO}/releases/download/${LATEST}"
BINARY_ARCHIVE="dewbu_${VERSION}_${OS}_${ARCH}.tar.gz"
SKILLS_ARCHIVE="dewbu-skills_${VERSION}.tar.gz"
CHECKSUM_FILE="dewbu_${VERSION}_checksums.txt"

echo "==> Downloading checksums..."
curl -fsSL "${BASE_URL}/${CHECKSUM_FILE}" -o "$TMP_DIR/checksums.txt"

echo "==> Downloading dewbu binary..."
curl -fsSL "${BASE_URL}/${BINARY_ARCHIVE}" -o "$TMP_DIR/${BINARY_ARCHIVE}"

echo "==> Downloading skills..."
curl -fsSL "${BASE_URL}/${SKILLS_ARCHIVE}" -o "$TMP_DIR/${SKILLS_ARCHIVE}"

# Verify checksums
echo "==> Verifying checksums..."
for archive in "$BINARY_ARCHIVE" "$SKILLS_ARCHIVE"; do
  EXPECTED=$(awk -v file="$archive" '$2 == file {print $1}' "$TMP_DIR/checksums.txt")
  ACTUAL=$(sha256_file "$TMP_DIR/$archive")
  if [[ -z "$EXPECTED" ]]; then
    echo "    ERROR: checksum entry not found for $archive" >&2
    exit 1
  fi
  if [[ "$EXPECTED" != "$ACTUAL" ]]; then
    echo "    ERROR: checksum mismatch for $archive" >&2
    echo "    Expected: $EXPECTED" >&2
    echo "    Actual:   $ACTUAL" >&2
    exit 1
  fi
  echo "    OK: $archive"
done

# Install binary
mkdir -p "$INSTALL_DIR"
tar -xzf "$TMP_DIR/${BINARY_ARCHIVE}" -C "$INSTALL_DIR" dewbu
chmod +x "$INSTALL_DIR/dewbu"
echo "    Installed: $INSTALL_DIR/dewbu"

# Install skills. The skills archive contains top-level skill directories
# such as dewbu-persona, dewbu-interview, dn-persona, and dn-interview.
TMP_SKILLS=$(mktemp -d)
tar -xzf "$TMP_DIR/${SKILLS_ARCHIVE}" -C "$TMP_SKILLS"

for dir in "${SKILLS_DIRS[@]}"; do
  mkdir -p "$dir"
  found_skills=0
  for skill in "$TMP_SKILLS"/*; do
    [[ -d "$skill" ]] || continue
    [[ -f "$skill/SKILL.md" ]] || continue
    found_skills=1
    skill_name=$(basename "$skill")
    rm -rf "$dir/$skill_name"
    cp -r "$skill" "$dir/$skill_name"
    echo "    Installed skill: $dir/$skill_name"
  done
  if [[ "$found_skills" -eq 0 ]]; then
    echo "    ERROR: no skill directories found in ${SKILLS_ARCHIVE}" >&2
    exit 1
  fi
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
