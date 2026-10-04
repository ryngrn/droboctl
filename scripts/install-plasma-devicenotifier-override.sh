#!/usr/bin/env bash
set -euo pipefail
branch=${PLASMA_BRANCH:-Plasma/6.6}
base="$HOME/.local/share/plasma/plasmoids/org.kde.plasma.devicenotifier"
mkdir -p "$base/contents/ui"
for f in main.qml FullRepresentation.qml DeviceItem.qml; do
  curl -fsSL "https://raw.githubusercontent.com/KDE/plasma-workspace/$branch/applets/devicenotifier/qml/$f" -o "$base/contents/ui/$f"
done
cat > "$base/metadata.json" <<"META"
{
  "KPackageStructure": "Plasma/Applet",
  "KPlugin": {
    "Id": "org.kde.plasma.devicenotifier",
    "Name": "Disks & Devices",
    "Description": "Notifications and access for devices",
    "Icon": "device-notifier",
    "License": "LGPL-2.0-or-later",
    "EnabledByDefault": true
  },
  "X-Plasma-API-Minimum-Version": "6.0"
}
META
python3 - "$base/contents/ui/DeviceItem.qml" <<"PY"
from pathlib import Path
import sys
p=Path(sys.argv[1]); s=p.read_text()
s=s.replace("    icon: deviceItem.deviceIcon\n", "    readonly property bool isDrobo: deviceItem.deviceDescription === \\\"Drobo\\\" || deviceItem.deviceDescription === \\\"Drobo 5D\\\"\n\n    icon: deviceItem.isDrobo ? \\\"drobo\\\" : deviceItem.deviceIcon\n")
s=s.replace("    title: deviceItem.deviceDescription\n", "    title: deviceItem.isDrobo ? \\\"Drobo 5D\\\" : deviceItem.deviceDescription\n")
needle="""        } else if (!deviceItem.deviceIsBusy) {\n            if (deviceItem.deviceFreeSpace > 0 && deviceItem.deviceSize > 0) {\n                return i18nc(\"@info:status Free disk space\", \"%1 free of %2\", deviceItem.deviceFreeSpaceText, deviceItem.deviceSizeText)\n            }\n"""
repl="""        } else if (!deviceItem.deviceIsBusy) {\n            if (deviceItem.isDrobo) {\n                return \"10.93 TB usable\"\n            }\n            if (deviceItem.deviceFreeSpace > 0 && deviceItem.deviceSize > 0) {\n                return i18nc(\"@info:status Free disk space\", \"%1 free of %2\", deviceItem.deviceFreeSpaceText, deviceItem.deviceSizeText)\n            }\n"""
if needle not in s: raise SystemExit("unsupported DeviceItem.qml; Plasma source changed")
s=s.replace(needle,repl).replace("\\\\\"", "\"")
p.write_text(s)
PY
qdbus6 org.kde.plasmashell /PlasmaShell org.kde.PlasmaShell.refreshCurrentShell >/dev/null 2>&1 || true
