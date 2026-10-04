package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ryngrn/droboctl/internal/device"
)

func mountCmd(args []string) error {
	fs := flag.NewFlagSet("mount", flag.ContinueOnError)
	explicit := fs.String("device", "", "SCSI generic device, e.g. /dev/sg3")
	mountpoint := fs.String("mountpoint", "/mnt/drobo", "mount point")
	if err := fs.Parse(args); err != nil {
		return err
	}

	d, err := device.One(*explicit)
	if err != nil {
		return err
	}
	if d.Block == "" {
		return fmt.Errorf("Drobo block device not found")
	}

	part, fsType, label, err := findDataPartition(d.Block)
	if err != nil {
		return err
	}
	if fsType != "hfsplus" {
		return fmt.Errorf("unsupported filesystem %q on %s", fsType, part)
	}

	if mp, ok := mountedAt(part); ok {
		fmt.Printf("%s is already mounted at %s\n", part, mp)
		return nil
	}

	if err := os.MkdirAll(*mountpoint, 0755); err != nil {
		return fmt.Errorf("create mountpoint: %w", err)
	}
	cmd := exec.Command("mount", "-t", "hfsplus", "-o", "ro", part, *mountpoint)
	if os.Geteuid() != 0 {
		cmd = exec.Command("sudo", "mount", "-t", "hfsplus", "-o", "ro", part, *mountpoint)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if label == "" {
		label = "Drobo"
	}
	fmt.Printf("Mounted %s (%s) read-only at %s\n", label, part, *mountpoint)
	return nil
}

func findDataPartition(block string) (part, fsType, label string, err error) {
	cmd := exec.Command("lsblk", "-lnpo", "NAME,FSTYPE,LABEL", block)
	out, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("lsblk: %w", err)
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		typ := fields[1]
		lbl := ""
		if len(fields) > 2 {
			lbl = strings.Join(fields[2:], " ")
		}
		if name == block {
			continue
		}
		if typ == "hfsplus" {
			return name, typ, lbl, nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", "", "", err
	}
	return "", "", "", fmt.Errorf("no supported Drobo data partition found under %s", block)
}

func mountedAt(part string) (string, bool) {
	b, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && filepath.Clean(f[0]) == filepath.Clean(part) {
			return f[1], true
		}
	}
	return "", false
}
