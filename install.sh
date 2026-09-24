#!/bin/sh
# install.sh: download, verify, and install the avochato CLI.
# Usage: curl -fsSL https://raw.githubusercontent.com/avochato/avochato-cli/master/install.sh | sh
# INSTALL_DIR picks the destination (default /usr/local/bin, sudo if needed); VERSION pins a release tag.

set -e

REPO="avochato/avochato-cli"
BINARY="avochato"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS. Download manually from https://github.com/$REPO/releases" >&2; exit 1 ;;
esac

if [ -n "$VERSION" ]; then
  BASE="https://github.com/$REPO/releases/download/$VERSION"
else
  BASE="https://github.com/$REPO/releases/latest/download"
fi
ASSET="${BINARY}_${OS}_${ARCH}.tar.gz"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading $ASSET..."
curl -fsSL "$BASE/$ASSET" -o "$TMP/$ASSET"
curl -fsSL "$BASE/checksums.txt" -o "$TMP/checksums.txt"

# Releases are signed with cosign (keyless, GitHub Actions OIDC); verify when cosign is installed.
if command -v cosign >/dev/null 2>&1; then
  echo "Verifying signature..."
  curl -fsSL "$BASE/checksums.txt.sigstore.json" -o "$TMP/checksums.txt.sigstore.json"
  cosign verify-blob "$TMP/checksums.txt" \
    --bundle "$TMP/checksums.txt.sigstore.json" \
    --certificate-identity-regexp "^https://github.com/$REPO/" \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com
else
  echo "cosign not found; skipping signature verification (checksum only)"
fi

echo "Verifying checksum..."
EXPECTED=$(grep " $ASSET\$" "$TMP/checksums.txt" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  echo "No checksum found for $ASSET" >&2; exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMP/$ASSET" | awk '{print $1}')
else
  ACTUAL=$(shasum -a 256 "$TMP/$ASSET" | awk '{print $1}')
fi
if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum mismatch for $ASSET" >&2; exit 1
fi

tar -xzf "$TMP/$ASSET" -C "$TMP"

SUDO=""
if [ ! -d "$INSTALL_DIR" ] || [ ! -w "$INSTALL_DIR" ]; then
  if [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  fi
fi
$SUDO mkdir -p "$INSTALL_DIR"
$SUDO install -m 755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"

echo "✓ avochato installed to $INSTALL_DIR/$BINARY"
echo "  Run: avochato login"
