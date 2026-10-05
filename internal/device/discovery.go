package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const droboUSBVendorID = "19b9"

var supportedUSBProductIDs = map[string]struct{}{
	"3444": {}, // Drobo 5D
}

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
		if !isSupportedUSBDevice(p) {
			continue
		}
		vendor := readTrim(filepath.Join(p, "device/vendor"))
		model := readTrim(filepath.Join(p, "device/model"))
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

func isSupportedUSBDevice(path string) bool {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	for p := filepath.Clean(real); ; p = filepath.Dir(p) {
		vendorID := strings.ToLower(readTrim(filepath.Join(p, "idVendor")))
		productID := strings.ToLower(readTrim(filepath.Join(p, "idProduct")))
		if vendorID != "" || productID != "" {
			_, supported := supportedUSBProductIDs[productID]
			return vendorID == droboUSBVendorID && supported
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
	}
	return false
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
