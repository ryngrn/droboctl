package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Device struct {
	SG     string
	Block  string
	Vendor string
	Model  string
}

func Discover() ([]Device, error) {
	paths, err := filepath.Glob("/sys/class/scsi_generic/sg*")
	if err != nil {
		return nil, err
	}
	var out []Device
	for _, p := range paths {
		vendor := readTrim(filepath.Join(p, "device/vendor"))
		model := readTrim(filepath.Join(p, "device/model"))
		v := strings.ToLower(vendor)
		m := strings.ToLower(model)
		if !strings.Contains(v, "drobo") && !strings.Contains(m, "drobo") &&
			!(strings.TrimSpace(vendor) == "TRUSTED") && !(strings.TrimSpace(vendor) == "USB 3.0") {
			continue
		}
		sg := "/dev/" + filepath.Base(p)
		block := ""
		entries, _ := os.ReadDir(filepath.Join(p, "device/block"))
		if len(entries) > 0 {
			block = "/dev/" + entries[0].Name()
		}
		out = append(out, Device{SG: sg, Block: block, Vendor: vendor, Model: model})
	}
	return out, nil
}

func One(explicit string) (Device, error) {
	if explicit != "" {
		return Device{SG: explicit}, nil
	}
	ds, err := Discover()
	if err != nil {
		return Device{}, err
	}
	if len(ds) == 0 {
		return Device{}, fmt.Errorf("no supported Drobo found")
	}
	if len(ds) > 1 {
		return Device{}, fmt.Errorf("multiple Drobos found; use --device /dev/sgX")
	}
	return ds[0], nil
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
