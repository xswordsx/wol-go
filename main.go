// Binary wol-go is an HTTP server that provides a simple web interface for sending a Wake-On-Lan
// magic packet to machines in the network.
package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	//go:embed pages/page.html
	htmlPage string

	//go:embed pages/ok.html
	okPage string
)

func main() {
	addr := flag.String("addr", ":8080", "Address on which the service will listen")
	flag.Parse()

	if addr == nil || *addr == "" {
		fmt.Fprintln(os.Stderr, "addr must be non-empty")
		os.Exit(1)
	}

	logger := log.New(os.Stdout, "[wol-go] ", log.Ltime|log.Lmicroseconds)

	logger.Println("Starting service on http://127.0.0.1:8080")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(htmlPage)) })
	mux.HandleFunc("POST /", handleWakeup)

	server := http.Server{
		Addr:           *addr,
		Handler:        mux,
		MaxHeaderBytes: 512,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("HTTP Server stopped unexpectedly: %v", err)
	}
	logger.Println("Exiting")
}

func handleWakeup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("cannot parse form: %v", err), http.StatusInternalServerError)
		return
	}
	var (
		ip    = r.Form.Get("ip")
		mac   = r.Form.Get("mac")
		ports = r.Form.Get("port")
	)
	if ip == "" || mac == "" || ports == "" {
		http.Error(w, "Missing parameter", http.StatusBadRequest)
		return
	}

	for _, port := range r.Form["port"] {
		if err := sendMagicPacket(ip, mac, port); err != nil {
			http.Error(
				w,
				fmt.Sprintf("could not send packet to %s:%s: %v", ip, port, err),
				http.StatusInternalServerError,
			)
			return
		}
	}

	w.Write([]byte(okPage))
}

func sendMagicPacket(ip string, mac string, port string) error {
	conn, err := net.Dial("udp", ip+":"+port)
	if err != nil {
		return fmt.Errorf("cannot dial network: %w", err)
	}
	defer conn.Close()

	var packet = [102]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	var macBytes = [6]byte{}
	for i, sval := range strings.Split(mac, ":") {
		val, err := strconv.ParseUint(sval, 16, 8)
		if err != nil {
			return fmt.Errorf("could not decode byte %q: %w", sval, err)
		}
		macBytes[i] = byte(val)
	}

	for i := 1; i <= 16; i++ {
		copy(packet[6*i:], macBytes[:])
	}

	n, err := conn.Write(packet[:])
	if err != nil {
		return fmt.Errorf("cannot write magic packet to addr: %w", err)
	}
	if n != 102 {
		return fmt.Errorf("written %d bytes instead of 102", n)
	}
	return nil
}
