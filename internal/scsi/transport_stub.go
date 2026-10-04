//go:build !linux

package scsi

import "fmt"

func Read(device string, cdb []byte, allocation int) ([]byte, error) {
	return nil, fmt.Errorf("live Drobo access is currently supported on Linux only")
}
