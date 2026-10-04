#!/usr/bin/env bash
set -euo pipefail
interval=${DROBO_AUTOMOUNT_INTERVAL:-2}
last_udi=""
while :; do
  dev=$(readlink -f /dev/disk/by-label/Drobo 2>/dev/null || true)
  if [[ -n "$dev" && -b "$dev" ]]; then
    props=$(udevadm info --query=property --name="$dev" 2>/dev/null || true)
    if grep -q "^ID_VENDOR_ID=19b9$" <<<"$props" && grep -q "^ID_MODEL_ID=3444$" <<<"$props"; then
      base=$(basename "$dev")
      udi="/org/freedesktop/UDisks2/block_devices/$base"
      if [[ "$udi" != "$last_udi" ]]; then
        kwriteconfig6 --file kded_device_automounterrc --group Devices --group "$udi" --key EverMounted true
        kwriteconfig6 --file kded_device_automounterrc --group Devices --group "$udi" --key ForceAttachAutomount true
        kwriteconfig6 --file kded_device_automounterrc --group Devices --group "$udi" --key ForceLoginAutomount false
        kwriteconfig6 --file kded_device_automounterrc --group Devices --group "$udi" --key LastNameSeen "Drobo 5D"
        kwriteconfig6 --file kded_device_automounterrc --group Devices --group "$udi" --key Icon drobo
        qdbus6 org.kde.kded6 /kded org.kde.kded6.reconfigure >/dev/null 2>&1 || true
        last_udi="$udi"
      fi
      if ! findmnt -rn -S "$dev" >/dev/null 2>&1; then
        udisksctl mount -b "$dev" >/dev/null 2>&1 || true
      fi
    fi
  else
    last_udi=""
  fi
  sleep "$interval"
done
