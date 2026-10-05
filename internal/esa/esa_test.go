package esa

import (
	"encoding/binary"
	"testing"
)

func TestProtocolVersion(t *testing.T) {
	b := make([]byte, 6)
	b[0], b[1] = 0x7a, ProtocolVersion
	binary.BigEndian.PutUint16(b[2:4], 2)
	b[4], b[5] = 0, 11

	maj, min, err := ParseProtocolVersion(b)
	if err != nil {
		t.Fatal(err)
	}
	if maj != 0 || min != 11 {
		t.Fatalf("got %d.%d, want 0.11", maj, min)
	}
}

func TestCapacity(t *testing.T) {
	const (
		free  = uint64(8_000_000_000_000)
		used  = uint64(2_000_000_000_000)
		total = uint64(10_000_000_000_000)
	)
	b := make([]byte, 60)
	b[0], b[1] = 0x7a, CapacityInfo
	binary.BigEndian.PutUint16(b[2:4], 56)
	binary.BigEndian.PutUint64(b[4:12], free)
	binary.BigEndian.PutUint64(b[12:20], used)
	binary.BigEndian.PutUint64(b[20:28], total)

	c, err := ParseCapacity(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.FreeProtected != free {
		t.Fatalf("free=%d", c.FreeProtected)
	}
	if c.UsedProtected != used {
		t.Fatalf("used=%d", c.UsedProtected)
	}
	if c.TotalProtected != total {
		t.Fatalf("total=%d", c.TotalProtected)
	}
	if c.FreeProtected+c.UsedProtected != c.TotalProtected {
		t.Fatal("free + used != total")
	}
}

func TestSlotInfo2(t *testing.T) {
	b := make([]byte, 12+108)
	b[0], b[1] = 0x7a, SlotInfo2
	binary.BigEndian.PutUint16(b[2:4], 116)
	b[8], b[9] = 1, 108
	e := b[12:]
	e[0], e[1] = 0, 3
	e[7] = 0x10
	copy(e[12:56], []byte("TEST HDD SATA"))
	copy(e[56:68], []byte("TEST1"))
	copy(e[68:92], []byte("TEST-SERIAL-0001"))
	binary.BigEndian.PutUint64(e[92:100], 3_000_000_000_000)
	binary.BigEndian.PutUint64(e[100:108], 2_500_000_000_000)

	slots, err := ParseSlotInfo2(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 1 || slots[0].Health() != "Good" || slots[0].Model != "TEST HDD" {
		t.Fatalf("unexpected slot: %+v", slots)
	}
	if slots[0].Serial != "TEST-SERIAL-0001" {
		t.Fatalf("serial=%q", slots[0].Serial)
	}
}
