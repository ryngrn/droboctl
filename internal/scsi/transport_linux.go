//go:build linux

package scsi

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	sgIO           = 0x2285
	sgDxferFromDev = -3
)

type sgIOHdr struct {
	InterfaceID    int32
	DxferDirection int32
	CmdLen         uint8
	MxSbLen        uint8
	IOVecCount     uint16
	DxferLen       uint32
	Dxferp         uintptr
	Cmdp           uintptr
	Sbp            uintptr
	Timeout        uint32
	Flags          uint32
	PackID         int32
	UsrPtr         uintptr
	Status         uint8
	MaskedStatus   uint8
	MsgStatus      uint8
	SbLenWr        uint8
	HostStatus     uint16
	DriverStatus   uint16
	Resid          int32
	Duration       uint32
	Info           uint32
}

func Read(device string, cdb []byte, allocation int) ([]byte, error) {
	f, err := os.OpenFile(device, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w (try sudo)", device, err)
	}
	defer f.Close()

	data := make([]byte, allocation)
	sense := make([]byte, 64)
	hdr := sgIOHdr{
		InterfaceID:    int32('S'),
		DxferDirection: sgDxferFromDev,
		CmdLen:         uint8(len(cdb)),
		MxSbLen:        uint8(len(sense)),
		DxferLen:       uint32(len(data)),
		Dxferp:         uintptr(unsafe.Pointer(&data[0])),
		Cmdp:           uintptr(unsafe.Pointer(&cdb[0])),
		Sbp:            uintptr(unsafe.Pointer(&sense[0])),
		Timeout:        5000,
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), sgIO, uintptr(unsafe.Pointer(&hdr)))
	if errno != 0 {
		return nil, fmt.Errorf("SG_IO: %w", errno)
	}
	if hdr.Status != 0 || hdr.HostStatus != 0 || hdr.DriverStatus != 0 {
		return nil, fmt.Errorf("SCSI command failed: status=0x%02x host=0x%04x driver=0x%04x sense=%x",
			hdr.Status, hdr.HostStatus, hdr.DriverStatus, sense[:hdr.SbLenWr])
	}
	n := len(data) - int(hdr.Resid)
	if n < 0 || n > len(data) {
		n = len(data)
	}
	return data[:n], nil
}
