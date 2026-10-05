package device

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSupportedUSBDevice(t *testing.T) {
	root := t.TempDir()
	usb := filepath.Join(root, "usb-device")
	sg := filepath.Join(usb, "host", "target", "scsi_generic", "sg3")
	if err := os.MkdirAll(sg, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usb, "idVendor"), []byte("19b9\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(usb, "idProduct"), []byte("3444\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !isSupportedUSBDevice(sg) {
		t.Fatal("expected Drobo 5D USB IDs to be supported")
	}

	if err := os.WriteFile(filepath.Join(usb, "idProduct"), []byte("ffff\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if isSupportedUSBDevice(sg) {
		t.Fatal("unexpected support for unknown USB product")
	}
}
