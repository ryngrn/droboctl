package main

import "testing"

func TestSerialForDisplay(t *testing.T) {
	if got := serialForDisplay("TEST-SERIAL-0001", false); got != "TEST-S…0001" {
		t.Fatalf("masked serial=%q", got)
	}
	if got := serialForDisplay("TEST-SERIAL-0001", true); got != "TEST-SERIAL-0001" {
		t.Fatalf("full serial=%q", got)
	}
	if got := serialForDisplay("SHORT", false); got != "redacted" {
		t.Fatalf("short serial=%q", got)
	}
}
