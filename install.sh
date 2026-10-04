#!/usr/bin/env bash
set -euo pipefail

REPO="ryngrn/droboctl"
VERSION="${DROBO_VERSION:-v0.1.0-rc1}"
DEST="/usr/local/bin/drobo"
RULE="/etc/udev/rules.d/99-droboctl.rules"
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

if command -v udevadm >/dev/null 2>&1; then
  printf '%s
' 'SUBSYSTEM=="scsi_generic", ATTRS{idVendor}=="19b9", TAG+="uaccess", MODE="0660"' | sudo tee "$RULE" >/dev/null
  sudo udevadm control --reload-rules
  sudo udevadm trigger --subsystem-match=scsi_generic || true
fi

echo "Installed drobo $VERSION to $DEST"
echo "Try: drobo status"
echo "If device permissions have not refreshed yet, unplug/replug the Drobo once."
