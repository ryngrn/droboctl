package esa

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	PageCode        = 0x3a
	ProtocolVersion = 0x06
	CapacityInfo    = 0x02
	StatusInfo      = 0x09
	SlotInfo2       = 0x35
)

type Capacity struct {
	FreeProtected    uint64
	UsedProtected    uint64
	TotalProtected   uint64
	TotalUnprotected uint64
	FreePassthrough  uint64
	UsedPassthrough  uint64
	TotalPassthrough uint64
}

type Slot struct {
	ID              uint8
	Status          uint8
	ErrorCount      uint16
	DiskStateRaw    uint32
	DiskType        uint8
	TemperatureC    uint8
	LifeRemaining   uint8
	RotationalSpeed uint8
	Model           string
	Firmware        string
	Serial          string
	TotalCapacity   uint64
	ManagedCapacity uint64
}

func (s Slot) Health() string {
	switch s.DiskStateRaw & 0x0f {
	case 0:
		return "Good"
	case 1:
		return "Healed"
	case 2:
		return "Warning"
	case 3:
		return "Failed"
	default:
		return "Unknown"
	}
}

func (s Slot) IsSSD() bool { return s.DiskType == 4 }

type Status struct {
	Word          uint64
	RelayoutCount uint32
	DiskPackWord  uint32
}

func (s Status) Severity() string {
	const redMask uint64 = 0x14c4187a
	const yellowMask uint64 = 0x36002300244
	if s.Word&redMask != 0 {
		return "Critical"
	}
	if s.Word&yellowMask != 0 {
		return "Warning"
	}
	return "Healthy"
}

func BuildModeSense10(subpage byte, allocation uint16) [10]byte {
	return [10]byte{0x5a, 0x00, PageCode, subpage, 0, 0, 0, byte(allocation >> 8), byte(allocation), 0}
}

func page(data []byte, subpage byte) ([]byte, error) {
	for i := 0; i+4 <= len(data) && i < 16; i++ {
		if data[i] == 0x7a && data[i+1] == subpage {
			n := int(binary.BigEndian.Uint16(data[i+2:i+4])) + 4
			if n > len(data)-i {
				n = len(data) - i
			}
			return data[i : i+n], nil
		}
	}
	return nil, fmt.Errorf("ESA page 0x%02x not found", subpage)
}

func ParseProtocolVersion(data []byte) (major, minor uint8, err error) {
	p, err := page(data, ProtocolVersion)
	if err != nil {
		return 0, 0, err
	}
	if len(p) < 6 {
		return 0, 0, errors.New("short protocol version page")
	}
	return p[4], p[5], nil
}

func ParseCapacity(data []byte) (Capacity, error) {
	p, err := page(data, CapacityInfo)
	if err != nil {
		return Capacity{}, err
	}
	if len(p) < 60 {
		return Capacity{}, fmt.Errorf("short capacity page: %d bytes", len(p))
	}
	return Capacity{
		FreeProtected:    binary.BigEndian.Uint64(p[4:12]),
		UsedProtected:    binary.BigEndian.Uint64(p[12:20]),
		TotalProtected:   binary.BigEndian.Uint64(p[20:28]),
		TotalUnprotected: binary.BigEndian.Uint64(p[28:36]),
		FreePassthrough:  binary.BigEndian.Uint64(p[36:44]),
		UsedPassthrough:  binary.BigEndian.Uint64(p[44:52]),
		TotalPassthrough: binary.BigEndian.Uint64(p[52:60]),
	}, nil
}

func ParseStatus(data []byte) (Status, error) {
	p, err := page(data, StatusInfo)
	if err != nil {
		return Status{}, err
	}
	if len(p) < 16 {
		return Status{}, fmt.Errorf("short status page: %d bytes", len(p))
	}
	return Status{
		Word:          uint64(binary.BigEndian.Uint32(p[4:8])),
		RelayoutCount: binary.BigEndian.Uint32(p[8:12]),
		DiskPackWord:  binary.BigEndian.Uint32(p[12:16]),
	}, nil
}

func ParseSlotInfo2(data []byte) ([]Slot, error) {
	p, err := page(data, SlotInfo2)
	if err != nil {
		return nil, err
	}
	if len(p) < 12 {
		return nil, fmt.Errorf("short slot page: %d bytes", len(p))
	}
	count := int(p[8])
	stride := int(p[9])
	if stride == 0 {
		stride = 108
	}
	if stride < 108 {
		return nil, fmt.Errorf("unexpected slot stride: %d", stride)
	}
	slots := make([]Slot, 0, count)
	for i := 0; i < count; i++ {
		start := 12 + i*stride
		if start+108 > len(p) {
			return nil, fmt.Errorf("slot %d truncated", i)
		}
		e := p[start : start+108]
		slots = append(slots, Slot{
			ID:              e[0],
			Status:          e[1],
			ErrorCount:      binary.BigEndian.Uint16(e[2:4]),
			DiskStateRaw:    binary.BigEndian.Uint32(e[4:8]),
			DiskType:        e[8],
			TemperatureC:    e[9],
			LifeRemaining:   e[10],
			RotationalSpeed: e[11],
			Model:           cleanASCII(e[12:56]),
			Firmware:        cleanASCII(e[56:68]),
			Serial:          cleanASCII(e[68:92]),
			TotalCapacity:   binary.BigEndian.Uint64(e[92:100]),
			ManagedCapacity: binary.BigEndian.Uint64(e[100:108]),
		})
	}
	return slots, nil
}

func cleanASCII(b []byte) string {
	end := len(b)
	for i, c := range b {
		if c == 0 || c == 0xff {
			end = i
			break
		}
	}
	s := strings.TrimSpace(string(b[:end]))
	if strings.HasSuffix(s, "SATA") {
		s = strings.TrimSpace(strings.TrimSuffix(s, "SATA"))
	}
	return s
}
