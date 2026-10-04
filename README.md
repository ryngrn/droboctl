# droboctl

**Turn your old Drobo into something useful again.**

droboctl is an open-source Linux utility for using and monitoring USB-connected Drobo storage without Drobo Dashboard.

The first target is the Drobo 5D. The enclosure keeps running its existing storage firmware; droboctl talks directly to the management channel the hardware already exposes over standard SCSI.

## Goal

Plug in a supported Drobo, run:

```bash
sudo drobo status
```

and get trustworthy capacity, array health, drive-bay information, rebuild state, and the Linux block device for the existing volume.

No Drobo Dashboard. No firmware modification. No kernel extension.

## Current status

Early development. Verified against a real Drobo 5D on Linux:

- USB VID:PID `19b9:3444`
- SuperSpeed USB 5 Gbps
- ESA protocol 0.11
- real protected/free/used capacity
- all five drive bays
- mSATA cache
- per-drive model, serial, firmware, health and capacity

The management path is read-only in v1. Destructive commands are intentionally out of scope.

## Quick install (Linux x86-64)

```bash
curl -fsSL https://raw.githubusercontent.com/ryngrn/droboctl/main/install.sh | bash
sudo drobo status
```

The installer uses the current development release while v0 is being validated on real hardware.

## Building

Requires Go.

```bash
go test ./...
GOOS=linux GOARCH=amd64 go build -o drobo ./cmd/drobo
```

On Linux, run it with permission to access the SCSI generic device:

```bash
sudo ./drobo status
```

Use `--device /dev/sgX` if more than one supported Drobo is attached.

## Prior art

The ESA protocol documentation and 5D validation work in [fetzu/ReDrobo](https://github.com/fetzu/ReDrobo) made this Linux implementation practical. ReDrobo is MIT licensed and documents the Drobo management channel in detail.

## Icon credit

The intended application icon is Bjango's replacement Drobo icon:

https://bjango.com/articles/droboicon/

Bjango publicly described the icon as open source. The project uses it with attribution and a link back to the original source.

## Safety

droboctl v1 only issues read-side SCSI management requests. It does not format disks, update firmware, modify the disk pack, resize volumes, or issue guessed write commands.

Your Drobo is still old storage hardware. Keep backups outside the Drobo.

## License

MIT. Not affiliated with Drobo, Inc. or its successors. "Drobo" is used only to identify compatible hardware.

## KDE / UDisks integration

On Plasma, droboctl can integrate a Drobo 5D into the normal removable-storage experience instead of creating a separate dashboard. The integration provides an original five-bay `drobo` icon, enables Plasma's removable-device automounter for the already-known Drobo only, keeps unknown removable devices from being automatically mounted, and supplies UDisks hints so the device is presented as **Drobo 5D**. The supplied udev rule also makes the HFS+ mount explicitly read-only.

Install the user-level portion:

```bash
./scripts/install-kde-integration.sh
```

Then install the system UDisks rule once:

```bash
sudo install -m 0644 integration/99-droboctl.rules /etc/udev/rules.d/99-droboctl.rules
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=block
```

The rule is deliberately restricted to USB VID:PID `19b9:3444`, filesystem label `Drobo`, and filesystem partitions. Plasma's `AutomountUnknownDevices` remains disabled; the known Drobo is forced to automount on attach through `kded_device_automounterrc`.
