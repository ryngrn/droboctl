#!/usr/bin/env bash
set -euo pipefail
interval=${DROBO_AUTOMOUNT_INTERVAL:-2}
while :; do
  dev=$(readlink -f /dev/disk/by-label/Drobo 2>/dev/null || true)
  if [[ -n "$dev" && -b "$dev" ]]; then
    props=$(udevadm info --query=property --name="$dev" 2>/dev/null || true)
    if grep -q "^ID_VENDOR_ID=19b9$" <<<"$props" && grep -q "^ID_MODEL_ID=3444$" <<<"$props"; then
      if ! findmnt -rn -S "$dev" >/dev/null 2>&1; then
        udisksctl mount -b "$dev" >/dev/null 2>&1 || true
      fi
    fi
  fi
  sleep "$interval"
done
