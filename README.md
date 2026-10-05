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
drobo status
```

The installer also adds a narrow udev rule for Drobo SCSI-generic devices so normal status reads can run without `sudo`. If permissions do not refresh immediately, unplug/replug the Drobo once.

For a focused health diagnostic:

```bash
drobo doctor
```

Drive serial numbers are masked in diagnostic output by default so logs are safer to share. Use `drobo doctor --show-serials` only when you specifically need the full identifiers.

To mount a detected HFS+ Drobo volume safely:

```bash
drobo mount
```

The mount command is intentionally read-only. `drobo status` also reports whether the data volume is currently mounted.

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

## Kudos

The ESA protocol documentation and 5D validation work in [fetzu/ReDrobo](https://github.com/fetzu/ReDrobo) made this Linux implementation practical. ReDrobo is MIT licensed and documents the Drobo management channel in detail. Great stuff!

Thanks to Bjango for creating some sweet icons for use: https://bjango.com/articles/droboicon/


## Safety & Next Steps

droboctl v1 only issues read-side SCSI management requests. 

In the future, it'll:
- format disks
- update firmware
- modify the disk pack
- resize volumes



## License

MIT. Not affiliated with Drobo, Inc. or its successors. "Drobo" is used only to identify compatible hardware.
