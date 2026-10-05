package integrate

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture() map[string]string {
	return map[string]string{"ID_BUS": "usb", "ID_VENDOR_ID": "19b9", "ID_MODEL_ID": "3444", "ID_VENDOR": "Drobo", "ID_MODEL": "5D", "ID_FS_UUID": "1234-5678"}
}
func manager(t *testing.T) (*Manager, map[string]string) {
	t.Helper()
	base := t.TempDir()
	values := map[string]string{}
	m := &Manager{Config: filepath.Join(base, "config"), Data: filepath.Join(base, "data"), State: filepath.Join(base, "state/integration.json"), VolumeInfo: func(string) (map[string]string, error) { return fixture(), nil }}
	m.Run = func(name string, a ...string) (string, error) {
		key := strings.Join(a[:len(a)-2], "|")
		if name == "kreadconfig6" {
			v, ok := values[key]
			if !ok {
				return missing, nil
			}
			return v, nil
		}
		if name == "kwriteconfig6" {
			if a[len(a)-1] == "--delete" {
				key = strings.Join(a[:len(a)-1], "|")
				delete(values, key)
			} else {
				key = strings.Join(a[:len(a)-1], "|")
				values[key] = a[len(a)-1]
			}
			return "", nil
		}
		return "true", nil
	}
	return m, values
}

// Fake KConfig commands retain file/group/key arguments exactly as KDE receives them.
func TestLifecycle(t *testing.T) {
	m, values := manager(t)
	udi := "/org/freedesktop/UDisks2/block_devices/sde2"
	old := filepath.Join(m.Data, "icons/hicolor/scalable/devices/drobo.svg")
	os.MkdirAll(filepath.Dir(old), 0755)
	os.WriteFile(old, []byte("previous icon"), 0644)
	if e := m.Install(udi, "/home/a path/drobo"); e != nil {
		t.Fatal(e)
	}
	first, _ := os.ReadFile(m.State)
	if e := m.Install(udi, "/home/a path/drobo"); e != nil {
		t.Fatal(e)
	}
	second, _ := os.ReadFile(m.State)
	if !bytes.Equal(first, second) {
		t.Fatal("backup changed on repeat install")
	}
	for _, k := range keys(udi) {
		v, e := m.read(k)
		if e != nil || v != k.After {
			t.Fatalf("wrong key %s: %q %v", k.Name, v, e)
		}
	}
	for _, name := range []string{"drobo.svg", "drobo-symbolic.svg"} {
		b, e := os.ReadFile(filepath.Join(m.Data, "icons/hicolor/scalable/devices", name))
		if e != nil {
			t.Fatal(e)
		}
		var doc interface{}
		if e = xml.Unmarshal(b, &doc); e != nil {
			t.Fatal(e)
		}
	}
	a, _ := os.ReadFile(filepath.Join(m.Data, "solid/actions/droboctl-status.desktop"))
	if !strings.Contains(string(a), "StorageVolume.uuid == '1234-5678'") || !strings.Contains(string(a), `"/home/a path/drobo" status`) {
		t.Fatal(string(a))
	}
	values["unrelated"] = "keep"
	if e := m.Uninstall(); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(old)
	if string(b) != "previous icon" || len(values) != 1 || values["unrelated"] != "keep" {
		t.Fatalf("restoration failed %v", values)
	}
	if e := m.Uninstall(); e != nil {
		t.Fatal(e)
	}
}
func TestPreserveExternalEdits(t *testing.T) {
	m, _ := manager(t)
	if e := m.Install("/org/freedesktop/UDisks2/block_devices/sde2", "/tmp/drobo"); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(m.Data, "icons/hicolor/scalable/devices/drobo.svg")
	os.WriteFile(path, []byte("user edit"), 0644)
	if e := m.Uninstall(); e == nil {
		t.Fatal("expected conflict")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "user edit" {
		t.Fatal("overwrote user edit")
	}
	if _, e := os.Stat(m.State); e != nil {
		t.Fatal("lost backup")
	}
}
func TestRejectNonDrobo(t *testing.T) {
	for _, key := range []string{"ID_BUS", "ID_VENDOR_ID", "ID_MODEL_ID", "ID_VENDOR", "ID_MODEL"} {
		p := fixture()
		p[key] = "other"
		if Matches(p) {
			t.Fatal(key)
		}
	}
	m, _ := manager(t)
	m.VolumeInfo = func(string) (map[string]string, error) { p := fixture(); p["ID_MODEL"] = "USB disk"; return p, nil }
	if e := m.Install("/org/freedesktop/UDisks2/block_devices/sde2", "/tmp/drobo"); e == nil {
		t.Fatal("accepted non-Drobo")
	}
	if _, e := os.Stat(m.State); !os.IsNotExist(e) {
		t.Fatal("wrote state for unknown device")
	}
}
func TestRule(t *testing.T) {
	b, _ := assets.ReadFile("assets/99-droboctl.rules")
	lines := strings.Split(string(b), "\n")
	blockRules := 0
	scsiRules := 0
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, `SUBSYSTEM=="block"`):
			blockRules++
			for _, match := range []string{`ENV{ID_BUS}=="usb"`, `ENV{ID_VENDOR_ID}=="19b9"`, `ENV{ID_MODEL_ID}=="3444"`, `ENV{ID_VENDOR}=="Drobo"`, `ENV{ID_MODEL}=="5D"`} {
				if !strings.Contains(l, match) {
					t.Fatal("unsafe block rule", l)
				}
			}
		case strings.HasPrefix(l, `SUBSYSTEM=="scsi_generic"`):
			scsiRules++
			for _, match := range []string{`ATTRS{vendor}=="Drobo*"`, `ATTRS{model}=="5D*"`, `GROUP="drobo"`, `MODE="0660"`} {
				if !strings.Contains(l, match) {
					t.Fatal("unsafe SCSI rule", l)
				}
			}
		}
	}
	if blockRules != 2 || scsiRules != 1 {
		t.Fatalf("unexpected rule counts: block=%d scsi=%d", blockRules, scsiRules)
	}
	for _, required := range []string{`ENV{UDISKS_AUTO}="1"`, `ENV{UDISKS_NAME}="Drobo 5D"`, `ENV{UDISKS_ICON_NAME}="drobo"`, `ENV{UDISKS_SYMBOLIC_ICON_NAME}="drobo-symbolic"`, `ENV{UDISKS_MOUNT_OPTIONS_DEFAULTS}="ro"`} {
		if !strings.Contains(string(b), required) {
			t.Fatal(required)
		}
	}
}
func TestNarrowAutomount(t *testing.T) {
	for _, k := range keys("known") {
		if k.Name == "AutomountOnPlugin" || k.Name == "AutomountOnLogin" || k.Name == "AutomountUnknownDevices" {
			if k.After != "false" {
				t.Fatal(k)
			}
		}
		if k.Name == "ForceAttachAutomount" && (len(k.Groups) != 2 || k.Groups[1] != "known" || k.After != "true") {
			t.Fatal(k)
		}
	}
}
func TestStaleCleanupGuard(t *testing.T) {
	for _, source := range []string{"/dev/sdd2 hfsplus ro,nosuid", "/dev/sde2 hfsplus ro", "/dev/sdd2 ext4 ro", "/dev/sdd2 hfsplus rw"} {
		called := false
		command := func(name string, a ...string) (string, error) {
			if name == "findmnt" {
				return source, nil
			}
			if name != "umount" || strings.Join(a, " ") != "-- /mnt/drobo" {
				t.Fatal(name, a)
			}
			called = true
			return "", nil
		}
		e := cleanupStale("/dev/sdd2", command, func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
		if source == "/dev/sdd2 hfsplus ro,nosuid" {
			if e != nil || !called {
				t.Fatal(e)
			}
		} else if e == nil || called {
			t.Fatal("unsafe stale cleanup", source)
		}
	}
}
