package esa

import (
	"encoding/hex"
	"testing"
)

func TestProtocolVersionLive5D(t *testing.T) {
	b, _ := hex.DecodeString("7a060002000bffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	maj, min, err := ParseProtocolVersion(b)
	if err != nil {
		t.Fatal(err)
	}
	if maj != 0 || min != 11 {
		t.Fatalf("got %d.%d, want 0.11", maj, min)
	}
}

func TestCapacityLive5D(t *testing.T) {
	b, _ := hex.DecodeString("7a020038000007648b6480000000028d55738000000009f1e0d80000000000000000000000000000000000000000000000000000000000000000ffffffff")
	c, err := ParseCapacity(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.FreeProtected != 8128416743424 {
		t.Fatalf("free=%d", c.FreeProtected)
	}
	if c.UsedProtected != 2806047277056 {
		t.Fatalf("used=%d", c.UsedProtected)
	}
	if c.TotalProtected != 10934464020480 {
		t.Fatalf("total=%d", c.TotalProtected)
	}
	if c.FreeProtected+c.UsedProtected != c.TotalProtected {
		t.Fatal("free + used != total")
	}
}

func TestSlotInfo2(t *testing.T) {
	b := make([]byte, 12+108)
	b[0], b[1] = 0x7a, 0x35
	b[2], b[3] = 0, 116
	b[8], b[9] = 1, 108
	e := b[12:]
	e[0], e[1] = 0, 3
	e[7] = 0x10
	copy(e[12:56], []byte("WDC WD30EZRX-00S SATA"))
	copy(e[56:68], []byte("80.00A80"))
	copy(e[68:92], []byte("WD-WCC4E0504081"))
	e[92], e[93], e[94], e[95], e[96], e[97], e[98], e[99] = 0, 0, 2, 0xba, 0xa1, 0x47, 0x60, 0
	slots, err := ParseSlotInfo2(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 1 || slots[0].Health() != "Good" || slots[0].Model != "WDC WD30EZRX-00S" {
		t.Fatalf("unexpected slot: %+v", slots)
	}
}
