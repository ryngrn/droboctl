#!/usr/bin/env bash
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
action=${1:-install}
source_pkg="$root/plasma/org.kmac.devicenotifier"
target="$HOME/.local/share/plasma/plasmoids/org.kmac.devicenotifier"
stock_override="$HOME/.local/share/plasma/plasmoids/org.kde.plasma.devicenotifier"
cfg="$HOME/.config/plasma-org.kde.plasma.desktop-appletsrc"
state="$HOME/.local/state/droboctl"
mkdir -p "$state"

install_direct_panel_widget() {
  python3 - "$cfg" <<'PYCFG'
from pathlib import Path
import re, sys
p=Path(sys.argv[1])
s=p.read_text()
# Remove stock Device Notifier from the embedded system tray.
s=re.sub(r'\n\[Containments\]\[28\]\[Applets\]\[35\]\[Applets\]\[43\]\n(?:[^\[]|\[(?!Containments))*?(?=\n\[Containments\]|\Z)', '\n', s, count=1, flags=re.S)
for key in ('extraItems','knownItems'):
    s=re.sub(rf'({key}=)([^\n]+)', lambda m: m.group(1)+','.join(x for x in m.group(2).split(',') if x not in ('org.kde.plasma.devicenotifier','org.kmac.devicenotifier')), s)
# Add the Kmac notifier as a direct top-panel applet if absent.
if 'plugin=org.kmac.devicenotifier' not in s:
    next_id=56
    while f'[Containments][28][Applets][{next_id}]' in s:
        next_id += 1
    insert=f'\n[Containments][28][Applets][{next_id}]\nimmutability=1\nplugin=org.kmac.devicenotifier\n'
    marker='\n[Containments][28][Applets][50]\n'
    s=s.replace(marker, insert+marker, 1)
    s=re.sub(r'AppletOrder=29;32;33;34;35(?:;\d+)?;50;51;52', f'AppletOrder=29;32;33;34;35;{next_id};50;51;52', s, count=1)
p.write_text(s)
PYCFG
}

case "$action" in
  install)
    [[ -f "$cfg" ]] || { echo "Plasma tray config not found: $cfg" >&2; exit 1; }
    if [[ ! -f "$state/plasma-appletsrc.before-devices-applet" ]]; then
      cp -a "$cfg" "$state/plasma-appletsrc.before-devices-applet"
    fi
    mkdir -p "$(dirname -- "$target")"
    if [[ -d "$target" ]]; then rm -rf "$target"; fi
    cp -a "$source_pkg" "$target"
    # Remove only our temporary user-level stock override; KDE's system plugin remains installed.
    if [[ -d "$stock_override" ]]; then mv "$stock_override" "$stock_override.disabled-$(date +%Y%m%d%H%M%S)"; fi
    install_direct_panel_widget
    ;;
  uninstall)
    if [[ -f "$state/plasma-appletsrc.before-devices-applet" ]]; then
      cp -a "$state/plasma-appletsrc.before-devices-applet" "$cfg"
    fi
    if [[ -d "$target" ]]; then mv "$target" "$target.disabled-$(date +%Y%m%d%H%M%S)"; fi
    ;;
  *) echo "Usage: $0 [install|uninstall]" >&2; exit 2 ;;
esac

kbuildsycoca6 --noincremental >/dev/null 2>&1 || true
qdbus6 org.kde.plasmashell /PlasmaShell org.kde.PlasmaShell.refreshCurrentShell >/dev/null 2>&1 || true
