// Binary wol-go is an HTTP server that provides a simple web interface for sending a Wake-On-Lan
// magic packet to machines in the network.
package main

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
)

type requestID int

const requestIDKey requestID = 0

var (
	//go:embed page.html
	htmlPage string
	cfg      config
)

type pageData struct {
	Machines []machine
	SentName string
}

func main() {
	cfgPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	if cfgPath == nil || *cfgPath == "" {
		fmt.Fprintln(os.Stderr, "-config must be non-empty")
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)).With("pid", os.Getpid())

	f, err := os.Open(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read config file: %v", err)
		os.Exit(1)
	}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot decode config file: %v", err)
		os.Exit(1)
	}
	f.Close()

	for i, m := range cfg.Machines {
		if err := m.validate(); err != nil {
			logger.Error("Bad machine configuration", "machine_id", i+1)
			os.Exit(1)
		}
	}

	tpl, err := template.New("main_page").Parse(htmlPage)
	if err != nil {
		logger.Error("Cannot parse HTML template", "error", err)
		os.Exit(1)
	}

	logger.Debug("Setting up service")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", getWakeUp(tpl, logger))
	mux.HandleFunc("POST /", postWakeUp(cfg.BroadcastAddress, cfg.Machines, tpl, logger))

	server := http.Server{
		Addr:           cfg.Address,
		Handler:        requestIDMiddleware(mux),
		MaxHeaderBytes: 512,
	}

	logger.Info("Starting HTTP server", "address", "http://"+cfg.Address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP Server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
	logger.Info("Exiting")
}

func magicPacket(mac string) ([]byte, error) {
	packet := [102]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	r := strings.NewReplacer(
		":", "",
		"-", "",
		" ", "",
	)
	clearMac := r.Replace(mac)
	macBytes, err := hex.DecodeString(clearMac)
	if err != nil {
		return nil, fmt.Errorf("invalid symbols in mac address: %w", err)
	}
	if len(macBytes) != 6 {
		return nil, errors.New("mac address size mismatch")
	}
	for i := 1; i <= 16; i++ {
		copy(packet[6*i:], macBytes[:])
	}

	return packet[:], nil
}

func sendMagicPacket(addr string, mac string) error {
	packet, err := magicPacket(mac)
	if err != nil {
		return fmt.Errorf("cannot generate magic packet: %w", err)
	}
	conn, err := net.Dial("udp4", addr)
	if err != nil {
		return fmt.Errorf("cannot dial network: %w", err)
	}
	defer conn.Close()

	n, err := conn.Write(packet)
	if err != nil {
		return fmt.Errorf("cannot write magic packet to addr: %w", err)
	}
	if n != 102 {
		return fmt.Errorf("written %d bytes instead of 102", n)
	}
	return nil
}
