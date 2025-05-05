package main

import (
	"slices"
	"testing"
)

func Test_magicPacket_parses_MAC_address(t *testing.T) {
	variations := []struct {
		name string
		mac  string
	}{
		{
			name: "Colon separators",
			mac:  "AA:BB:CC:DD:EE:FF",
		},
		{
			name: "Dash separators",
			mac:  "AA-BB-CC-DD-EE-FF",
		},
		{
			name: "Case insensitive",
			mac:  "AA-Bb-cc-Dd-eE-FF",
		},
	}
	for _, v := range variations {
		t.Run(v.name, func(t *testing.T) {
			_, err := magicPacket(v.mac)
			if err != nil {
				t.Fatalf("expected packet creation to succeed got '%v'", err)
			}
		})
	}
}

func Test_magicPacket_fails_on_bad_mac_address(t *testing.T) {
	variations := []struct {
		name string
		mac  string
	}{
		{
			name: "Incomplete address",
			mac:  "AA:BB:CC:DD:EE:",
		},
		{
			name: "Non-hex characters",
			mac:  "SO:ME:WE:IR:DS:TR",
		},
		{
			name: "Too big of an address",
			mac:  "AA:BB:CC:DD:EE:FF:00:11:22",
		},
	}
	for _, v := range variations {
		t.Run(v.name, func(t *testing.T) {
			_, err := magicPacket(v.mac)
			if err == nil {
				t.Fatal("expected func to error got nil")
			}
		})
	}
}

func Test_magicPacket_returns_correct_size_packet(t *testing.T) {
	mac := "AA:BB:CC:DD:EE:FF"
	packet, err := magicPacket(mac)
	if err != nil {
		t.Fatalf("expected packet creation to succeed got '%v'", err)
	}
	if len(packet) != 102 {
		t.Fatalf("expected packet to be of size 102 got %d", len(packet))
	}
}

func Test_magicPacket_returns_correct_data(t *testing.T) {
	mac := "01:23:45:67:89:ab"
	packet, err := magicPacket(mac)
	if err != nil {
		t.Fatalf("expected packet creation to succeed got '%v'", err)
	}

	t.Log("Testing padding")
	for i, b := range packet[:6] {
		if b != 0xff {
			t.Fatalf("expected byte #%d to be 0xff got 0x%x", i, b)
		}
	}

	t.Log("Testing MAC address repetition")
	macBytes := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB}
	for i := 1; i < 16; i++ {
		packetClip := packet[6*i : 6*(i+1)]
		if slices.Compare(packetClip, macBytes) != 0 {
			t.Fatalf(
				"expected packet bytes [%d:%d] to equal %v got %v",
				6*i, 6*(i+1), macBytes, packetClip,
			)
		}
	}
}
