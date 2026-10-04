#!/usr/bin/env bash
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
action=${1:-install}
source_pkg="$root/plasma/org.kmac.devicenotifier"
target="$HOME/.local/share/plasma/plasmoids/org.kmac.devicenotifier"
cfg="$HOME/.config/plasma-org.kde.plasma.desktop-appletsrc"
state="$HOME/.local/state/droboctl"
mkdir -p "$state"

rewrite_tray() {
  local from=$1 to=$2
  python3 - "$cfg" "$from" "$to" <<PY
from pathlib import Path
import sys
p=Path(sys.argv[1]); old=sys.argv[2]; new=sys.argv[3]
s=p.read_text()
if old not in s and new not in s:
    raise SystemExit(f"Neither {old} nor {new} is present in Plasma tray config")
s=s.replace(f"plugin={old}", f"plugin={new}", 1)
s=s.replace(old, new)
p.write_text(s)
PY
}

case "$action" in
  install)
    [[ -f "$cfg" ]] || { echo "Plasma tray config not found: $cfg" >&2; exit 1; }
    if [[ ! -f "$state/plasma-appletsrc.before-devices-applet" ]]; then
      cp -a "$cfg" "$state/plasma-appletsrc.before-devices-applet"
    fi
    mkdir -p "$(dirname -- "$target")"
    if [[ -d "$target" ]]; then mv "$target" "$target.backup-$(date +%Y%m%d%H%M%S)"; fi
    cp -a "$source_pkg" "$target"
    rewrite_tray org.kde.plasma.devicenotifier org.kmac.devicenotifier
    ;;
  uninstall)
    rewrite_tray org.kmac.devicenotifier org.kde.plasma.devicenotifier
    if [[ -d "$target" ]]; then mv "$target" "$target.disabled-$(date +%Y%m%d%H%M%S)"; fi
    ;;
  *) echo "Usage: $0 [install|uninstall]" >&2; exit 2 ;;
esac

kbuildsycoca6 --noincremental >/dev/null 2>&1 || true
qdbus6 org.kde.plasmashell /PlasmaShell org.kde.PlasmaShell.refreshCurrentShell >/dev/null 2>&1 || true
