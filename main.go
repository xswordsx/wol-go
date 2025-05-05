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
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

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

	logger := log.New(os.Stdout, "[wol-go] ", log.Ltime|log.Lmicroseconds)

	f, err := os.Open(*cfgPath)
	if err != nil {
		log.Fatalf("Cannot read config file: %v", err)
	}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("Cannot decode config file: %v", err)
	}
	f.Close()

	for i, m := range cfg.Machines {
		if err := m.validate(); err != nil {
			log.Fatalf("Machine #%d: %v", i+1, err)
		}
	}

	tpl, err := template.New("main_page").Parse(htmlPage)
	if err != nil {
		log.Fatalf("Cannot parse HTML template: %v", err)
	}

	logger.Printf("Starting service on http://%s", cfg.Address)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if err := tpl.Execute(w, pageData{Machines: cfg.Machines}); err != nil {
			log.Printf("Cannot execute template: %v", err)
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("POST /", handleWakeup(cfg.BroadcastAddress, cfg.Machines, tpl))

	server := http.Server{
		Addr:           cfg.Address,
		Handler:        mux,
		MaxHeaderBytes: 512,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("HTTP Server stopped unexpectedly: %v", err)
	}
	logger.Println("Exiting")
}

func handleWakeup(broadcast string, machines []machine, tpl *template.Template) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Cannot parse form: %v", err), http.StatusInternalServerError)
			return
		}
		i, err := strconv.Atoi(r.FormValue("machine"))
		if err != nil {
			http.Error(w, fmt.Sprintf("Cannot parse machine id: %v", err), http.StatusInternalServerError)
			return
		}
		if i < 0 || i >= len(machines) {
			http.Error(w, "Machine id out of range", http.StatusInternalServerError)
			return
		}

		mach := machines[i]
		for _, port := range mach.Ports {
			err := sendMagicPacket(fmt.Sprintf("%s:%d", broadcast, port), mach.MACAddress)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to send magic packet to machine: %v", err), http.StatusInternalServerError)
				return
			}
		}

		err = tpl.Execute(w, pageData{Machines: machines, SentName: mach.Name})
		if err != nil {
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			return
		}
	}
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
