#!/usr/bin/env bash
set -euo pipefail

REPO="ryngrn/droboctl"
VERSION="${DROBO_VERSION:-v0.0.2-dev}"
DEST="/usr/local/bin/drobo"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

case "$(uname -m)" in
  x86_64|amd64) asset="drobo-linux-amd64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

url="https://github.com/$REPO/releases/download/$VERSION/$asset"
curl -fL "$url" -o "$TMP/drobo"
chmod +x "$TMP/drobo"
sudo install -m 0755 "$TMP/drobo" "$DEST"

echo "Installed drobo $VERSION to $DEST"
echo "Run: sudo drobo status"
