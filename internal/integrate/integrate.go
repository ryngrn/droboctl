// Package integrate installs native KDE/UDisks metadata without volume writes.
package integrate

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed assets/*
var assets embed.FS

const rulePath = "/etc/udev/rules.d/99-droboctl.rules"
const missing = "__droboctl_absent_39bd__"

type Key struct {
	File                string
	Groups              []string
	Name, Before, After string
}
type File struct {
	Path          string
	Before, After []byte
	Existed       bool
}
type State struct {
	Keys  []Key
	Files []File
	UDI   string
}
type Manager struct {
	Config, Data, State string
	Run                 func(string, ...string) (string, error)
	VolumeInfo          func(string) (map[string]string, error)
}

func run(name string, args ...string) (string, error) {
	b, e := exec.Command(name, args...).CombinedOutput()
	if e != nil {
		return string(b), fmt.Errorf("%s: %w: %s", name, e, b)
	}
	return strings.TrimSpace(string(b)), nil
}
func New() (*Manager, error) {
	h, e := os.UserHomeDir()
	if e != nil {
		return nil, e
	}
	dir := func(env, fallback string) string {
		if p := os.Getenv(env); p != "" {
			return p
		}
		return filepath.Join(h, fallback)
	}
	return &Manager{dir("XDG_CONFIG_HOME", ".config"), dir("XDG_DATA_HOME", ".local/share"), filepath.Join(dir("XDG_STATE_HOME", ".local/state"), "droboctl/integration.json"), run, volumeInfo}, nil
}
func (m *Manager) args(k Key) []string {
	a := []string{"--file", filepath.Join(m.Config, k.File)}
	for _, g := range k.Groups {
		a = append(a, "--group", g)
	}
	return append(a, "--key", k.Name)
}
func (m *Manager) read(k Key) (string, error) {
	return m.Run("kreadconfig6", append(m.args(k), "--default", missing)...)
}
func (m *Manager) write(k Key, v string) error {
	a := m.args(k)
	if v == missing {
		a = append(a, "--delete")
	} else {
		a = append(a, v)
	}
	_, e := m.Run("kwriteconfig6", a...)
	return e
}
func Matches(p map[string]string) bool {
	return p["ID_BUS"] == "usb" && p["ID_VENDOR_ID"] == "19b9" && p["ID_MODEL_ID"] == "3444" && p["ID_VENDOR"] == "Drobo" && p["ID_MODEL"] == "5D"
}
func properties(path string) (map[string]string, error) {
	b, e := os.ReadFile(path)
	p := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "E:") {
			k, v, ok := strings.Cut(line[2:], "=")
			if ok {
				p[k] = v
			}
		}
	}
	return p, e
}

// UDisks/Solid uses the live block object path, not the filesystem UUID, as UDI.
// Discover through sysfs + udev database; device nodes may be sandbox-hidden.
func Volumes() ([]string, error) {
	paths, e := filepath.Glob("/sys/class/block/*")
	if e != nil {
		return nil, e
	}
	var out []string
	for _, p := range paths {
		d, e := os.ReadFile(filepath.Join(p, "dev"))
		if e != nil {
			continue
		}
		props, e := properties("/run/udev/data/b" + strings.TrimSpace(string(d)))
		if e == nil && Matches(props) && props["ID_FS_USAGE"] == "filesystem" && props["ID_PART_ENTRY_TYPE"] != "c12a7328-f81f-11d2-ba4b-00a0c93ec93b" {
			out = append(out, "/org/freedesktop/UDisks2/block_devices/"+filepath.Base(p))
		}
	}
	return out, nil
}
func volumeInfo(udi string) (map[string]string, error) {
	if !regexp.MustCompile(`^/org/freedesktop/UDisks2/block_devices/[A-Za-z0-9_]+$`).MatchString(udi) {
		return nil, fmt.Errorf("invalid UDI")
	}
	d, e := os.ReadFile("/sys/class/block/" + filepath.Base(udi) + "/dev")
	if e != nil {
		return nil, e
	}
	return properties("/run/udev/data/b" + strings.TrimSpace(string(d)))
}
func keys(udi string) []Key {
	k := func(file string, g []string, n, v string) Key { return Key{File: file, Groups: g, Name: n, After: v} }
	return []Key{k("kded5rc", []string{"Module-device_automounter"}, "autoload", "true"), k("kded_device_automounterrc", []string{"General"}, "AutomountEnabled", "true"), k("kded_device_automounterrc", []string{"General"}, "AutomountOnLogin", "false"), k("kded_device_automounterrc", []string{"General"}, "AutomountOnPlugin", "false"), k("kded_device_automounterrc", []string{"General"}, "AutomountUnknownDevices", "false"), k("kded_device_automounterrc", []string{"Devices", udi}, "ForceAttachAutomount", "true"), k("kded_device_automounterrc", []string{"Devices", udi}, "ForceLoginAutomount", "false")}
}
func (m *Manager) save(s State) error {
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(m.State), 0700); e != nil {
		return e
	}
	p := m.State + ".tmp"
	if e = os.WriteFile(p, b, 0600); e != nil {
		return e
	}
	return os.Rename(p, m.State)
}
func (m *Manager) load() (State, error) {
	var s State
	b, e := os.ReadFile(m.State)
	if e == nil {
		e = json.Unmarshal(b, &s)
	}
	return s, e
}
func (m *Manager) Install(udi, binary string) error {
	s, e := m.load()
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if e == nil && s.UDI != udi {
		return fmt.Errorf("installed for %s; uninstall before selecting another volume", s.UDI)
	}
	if errors.Is(e, os.ErrNotExist) {
		s.UDI = udi
		s.Keys = keys(udi)
		for i := range s.Keys {
			s.Keys[i].Before, e = m.read(s.Keys[i])
			if e != nil {
				return e
			}
		}
		for _, name := range []string{"drobo.svg", "drobo-symbolic.svg"} {
			b, _ := assets.ReadFile("assets/" + name)
			s.Files = append(s.Files, File{Path: filepath.Join(m.Data, "icons/hicolor/scalable/devices", name), After: b})
		}
		// A native Devices action opens the existing read-only CLI in Konsole.
		// UUID predicate prevents offering management on unrelated storage volumes.
		p, e := m.VolumeInfo(udi)
		if e != nil || !Matches(p) || !regexp.MustCompile(`^[A-Za-z0-9-]+$`).MatchString(p["ID_FS_UUID"]) {
			return fmt.Errorf("cannot verify selected Drobo filesystem")
		}

		action := fmt.Sprintf("[Desktop Entry]\nType=Service\nX-KDE-Solid-Predicate=StorageVolume.uuid == '%s'\nActions=status;\n\n[Desktop Action status]\nName=Show Drobo health\nIcon=drobo\nExec=konsole --hold -e %s status\n", p["ID_FS_UUID"], desktopQuote(binary))
		s.Files = append(s.Files, File{Path: filepath.Join(m.Data, "solid/actions/droboctl-status.desktop"), After: []byte(action)})
		for i := range s.Files {
			b, e := os.ReadFile(s.Files[i].Path)
			if e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
			s.Files[i].Before = b
			s.Files[i].Existed = e == nil
		}
		if e = m.save(s); e != nil {
			return e
		}
	}
	for _, k := range s.Keys {
		v, e := m.read(k)
		if e != nil {
			return e
		}
		if v != k.Before && v != k.After {
			return fmt.Errorf("refusing externally changed key %s", k.Name)
		}
		if e = m.write(k, k.After); e != nil {
			return e
		}
	}
	for _, f := range s.Files {
		b, e := os.ReadFile(f.Path)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if e == nil && !bytes.Equal(b, f.Before) && !bytes.Equal(b, f.After) {
			return fmt.Errorf("refusing externally changed file %s", f.Path)
		}
		if e = os.MkdirAll(filepath.Dir(f.Path), 0755); e != nil {
			return e
		}
		if e = os.WriteFile(f.Path, f.After, 0644); e != nil {
			return e
		}
	}
	return nil
}
func desktopQuote(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "$", "\\$", "`", "\\`", "%", "%%")
	return "\"" + r.Replace(s) + "\""
}
func (m *Manager) Uninstall() error {
	s, e := m.load()
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	var conflicts []string
	for _, k := range s.Keys {
		v, e := m.read(k)
		if e != nil {
			return e
		}
		if v == k.After {
			if e = m.write(k, k.Before); e != nil {
				return e
			}
		} else if v != k.Before {
			conflicts = append(conflicts, k.Name)
		}
	}
	for _, f := range s.Files {
		b, e := os.ReadFile(f.Path)
		if errors.Is(e, os.ErrNotExist) && !f.Existed {
			continue
		}
		if e != nil {
			return e
		}
		if bytes.Equal(b, f.After) {
			if f.Existed {
				e = os.WriteFile(f.Path, f.Before, 0644)
			} else {
				e = os.Remove(f.Path)
			}
			if e != nil {
				return e
			}
		} else if !bytes.Equal(b, f.Before) {
			conflicts = append(conflicts, f.Path)
		}
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("preserved external changes; retained backups: %v", conflicts)
	}
	return os.Remove(m.State)
}
func (m *Manager) Refresh() error {
	if _, e := exec.LookPath("gtk-update-icon-cache"); e == nil {
		_, _ = m.Run("gtk-update-icon-cache", "--force", "--ignore-theme-index", filepath.Join(m.Data, "icons/hicolor"))
	}
	_, _ = m.Run("kbuildsycoca6", "--noincremental")
	_, e := m.Run("qdbus6", "org.kde.kded6", "/kded", "org.kde.kded6.unloadModule", "device_automounter")
	if e != nil {
		return e
	}
	v, e := m.read(Key{File: "kded5rc", Groups: []string{"Module-device_automounter"}, Name: "autoload"})
	if e != nil {
		return e
	}
	if v == "true" {
		_, e = m.Run("qdbus6", "org.kde.kded6", "/kded", "org.kde.kded6.loadModule", "device_automounter")
	}
	return e
}
func System(install bool) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("root required only for udev: sudo drobo integrate install --system")
	}
	want, _ := assets.ReadFile("assets/99-droboctl.rules")
	b, e := os.ReadFile(rulePath)
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	const backup = "/var/lib/droboctl/udev-integration.json"
	var state File
	raw, stateErr := os.ReadFile(backup)
	if stateErr == nil {
		if e = json.Unmarshal(raw, &state); e != nil {
			return e
		}
		if state.Path != rulePath {
			return fmt.Errorf("invalid udev backup path")
		}
	} else if !errors.Is(stateErr, os.ErrNotExist) {
		return stateErr
	}
	if install {
		if errors.Is(stateErr, os.ErrNotExist) {
			state = File{Path: rulePath, Before: b, After: want, Existed: e == nil}
			raw, _ = json.MarshalIndent(state, "", "  ")
			if e = os.MkdirAll(filepath.Dir(backup), 0700); e != nil {
				return e
			}
			if e = os.WriteFile(backup, raw, 0600); e != nil {
				return e
			}
		} else if !bytes.Equal(b, state.After) && !bytes.Equal(b, state.Before) {
			return fmt.Errorf("preserving externally changed rule %s", rulePath)
		}
		if e = os.WriteFile(rulePath, want, 0644); e != nil {
			return e
		}
	} else {
		if errors.Is(stateErr, os.ErrNotExist) {
			return nil
		}
		if !bytes.Equal(b, state.After) && !bytes.Equal(b, state.Before) {
			return fmt.Errorf("preserving externally changed rule %s and backup", rulePath)
		}
		if state.Existed {
			e = os.WriteFile(rulePath, state.Before, 0644)
		} else if e == nil {
			e = os.Remove(rulePath)
		} else {
			e = nil
		}
		if e != nil {
			return e
		}
		if e = os.Remove(backup); e != nil {
			return e
		}
	}
	if _, e = run("udevadm", "control", "--reload-rules"); e != nil {
		return e
	}
	paths, _ := filepath.Glob("/sys/class/block/*")
	for _, p := range paths {
		d, e := os.ReadFile(filepath.Join(p, "dev"))
		if e != nil {
			continue
		}
		props, e := properties("/run/udev/data/b" + strings.TrimSpace(string(d)))
		if e == nil && Matches(props) {
			if _, e = run("udevadm", "trigger", "--action=change", p); e != nil {
				return e
			}
		}
	}
	_, e = run("udevadm", "settle", "--timeout=10")
	return e
}

// cleanupStale never targets a desktop mount and refuses a still-present block device.
func cleanupStale(expected string, command func(string, ...string) (string, error), stat func(string) (os.FileInfo, error)) error {
	if !regexp.MustCompile(`^/dev/sd[a-z]+[0-9]+$`).MatchString(expected) {
		return fmt.Errorf("invalid expected stale source")
	}
	out, e := command("findmnt", "-rn", "--mountpoint", "/mnt/drobo", "-o", "SOURCE,FSTYPE,OPTIONS")
	if e != nil {
		if out == "" {
			return nil
		}
		return e
	}
	fields := strings.Fields(out)
	if len(fields) != 3 || fields[0] != expected || fields[1] != "hfsplus" || !strings.Contains(","+fields[2]+",", ",ro,") {
		return fmt.Errorf("refusing unexpected /mnt/drobo mount: %s", out)
	}
	if _, e = stat("/sys/class/block/" + filepath.Base(expected)); !errors.Is(e, os.ErrNotExist) {
		return fmt.Errorf("refusing to unmount a present or unverifiable device")
	}
	_, e = command("umount", "--", "/mnt/drobo")
	return e
}
func Command(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: drobo integrate install|status|uninstall [--system]")
	}
	f := flag.NewFlagSet("integrate", flag.ContinueOnError)
	system := f.Bool("system", false, "install/remove only the root-owned udev rule")
	stale := f.String("stale-source", "", "with --system install: remove only a stale read-only /mnt/drobo mount from this vanished device")
	if e := f.Parse(args[1:]); e != nil {
		return e
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if *system {
		switch args[0] {
		case "install":
			if e := System(true); e != nil {
				return e
			}
			if *stale != "" {
				return cleanupStale(*stale, run, os.Stat)
			}
			return nil
		case "uninstall":
			return System(false)
		default:
			return fmt.Errorf("--system requires install or uninstall")
		}
	}
	if *stale != "" {
		return fmt.Errorf("--stale-source requires --system install")
	}
	if args[0] == "uninstall" && os.Geteuid() == 0 {
		return fmt.Errorf("run user uninstall without sudo")
	}
	m, e := New()
	if e != nil {
		return e
	}
	switch args[0] {
	case "install":
		if os.Geteuid() == 0 {
			return fmt.Errorf("run user integration as the desktop user, without sudo")
		}
		v, e := Volumes()
		if e != nil {
			return e
		}
		if len(v) != 1 {
			return fmt.Errorf("need exactly one connected Drobo filesystem, found %d", len(v))
		}
		b, e := os.Executable()
		if e != nil {
			return e
		}
		if e = m.Install(v[0], b); e != nil {
			return e
		}
		if e = m.Refresh(); e != nil {
			return fmt.Errorf("installed; live KDE reload failed: %w", e)
		}
		fmt.Println("User icons, native health action, and Drobo-only attach mounting installed. Root udev rule is a separate step.")
		return nil
	case "uninstall":
		if e = m.Uninstall(); e != nil {
			return e
		}
		return m.Refresh()
	case "status":
		s, e := m.load()
		if e == nil {
			fmt.Println("Managed volume:", s.UDI)
			for _, k := range s.Keys {
				v, e := m.read(k)
				if e != nil {
					return e
				}
				fmt.Printf("%s %v %s=%s\n", k.File, k.Groups, k.Name, v)
			}
		} else if errors.Is(e, os.ErrNotExist) {
			fmt.Println("User integration not installed")
		} else {
			return e
		}
		want, _ := assets.ReadFile("assets/99-droboctl.rules")
		b, _ := os.ReadFile(rulePath)
		fmt.Println("Exact system rule installed:", bytes.Equal(want, b))
		v, e := Volumes()
		if e != nil {
			return e
		}
		for _, u := range v {
			out, e := m.Run("udisksctl", "info", "-b", "/dev/"+filepath.Base(u))
			fmt.Println(out)
			if e != nil {
				return e
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown integration command %q", args[0])
	}
}
