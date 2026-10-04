#!/usr/bin/env bash
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
icons="$HOME/.local/share/icons/hicolor/scalable/devices"
mkdir -p "$icons"
install -m 0644 "$root/icons/drobo.svg" "$icons/drobo.svg"
install -m 0644 "$root/icons/drobo-symbolic.svg" "$icons/drobo-symbolic.svg"
command -v gtk-update-icon-cache >/dev/null && gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" >/dev/null 2>&1 || true
cfg="$HOME/.config/kded_device_automounterrc"
[[ -f "$cfg" ]] && cp -a "$cfg" "$cfg.droboctl-backup" || true
kwriteconfig6 --file kded_device_automounterrc --group General --key AutomountEnabled true
kwriteconfig6 --file kded_device_automounterrc --group General --key AutomountOnLogin false
kwriteconfig6 --file kded_device_automounterrc --group General --key AutomountOnPlugin true
kwriteconfig6 --file kded_device_automounterrc --group General --key AutomountUnknownDevices false
kwriteconfig6 --file kded_device_automounterrc --group Devices --group /org/freedesktop/UDisks2/block_devices/sde2 --key EverMounted true
kwriteconfig6 --file kded_device_automounterrc --group Devices --group /org/freedesktop/UDisks2/block_devices/sde2 --key ForceAttachAutomount true
qdbus6 org.kde.kded6 /kded org.kde.kded6.setModuleAutoloading device_automounter true >/dev/null 2>&1 || true
qdbus6 org.kde.kded6 /kded org.kde.kded6.loadModule device_automounter >/dev/null 2>&1 || true
qdbus6 org.kde.kded6 /kded org.kde.kded6.reconfigure >/dev/null 2>&1 || true
printf 'Drobo KDE user integration installed.\n'
printf 'Install the UDisks identity rule once with sudo; see README.\n'
