package main

import (
	"encoding/hex"
	"errors"
	"strings"
)

type config struct {
	Address          string    `json:"address,omitempty"`
	BroadcastAddress string    `json:"broadcast,omitempty"`
	Machines         []machine `json:"machines,omitempty"`
}

type machine struct {
	Name       string   `json:"name,omitempty"`
	MACAddress string   `json:"mac,omitempty"`
	Ports      []uint16 `json:"ports,omitempty"`
}

func (m *machine) validate() error {
	if m == nil {
		return errors.New("machine entry is nil")
	}
	if len(m.Ports) == 0 {
		return errors.New("missing ports")
	}

	r := strings.NewReplacer(
		":", "",
		"-", "",
		" ", "",
	)
	clearMac := r.Replace(m.MACAddress)
	macBytes, err := hex.DecodeString(clearMac)
	if err != nil {
		return errors.New("invalid symbols in mac address")
	}
	if len(macBytes) != 6 {
		return errors.New("mac address size mismatch")
	}

	return nil
}
