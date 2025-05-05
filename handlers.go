package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func getWakeUp(tpl *template.Template, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger = withRequestFields(logger, r)
		logger.Debug("Received request")
		defer func() { logger.Debug("Response sent", "took", time.Since(start)) }()

		if err := tpl.Execute(w, pageData{Machines: cfg.Machines}); err != nil {
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			logger.Error("Cannot execute template", "error", err)
			return
		}
	}
}

func postWakeUp(broadcastIP string, machines []machine, tpl *template.Template, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger = withRequestFields(logger, r)
		logger.Debug("Received request")
		defer func() { logger.Debug("Response sent", "took", time.Since(start)) }()

		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Cannot parse form: %v", err), http.StatusInternalServerError)
			logger.Error("Cannot parse form", "error", err)
			return
		}
		i, err := strconv.Atoi(r.FormValue("machine"))
		if err != nil {
			http.Error(w, fmt.Sprintf("Cannot parse machine id: %v", err), http.StatusInternalServerError)
			logger.Error(
				"Cannot parse machine id",
				"error", err,
				"machine_id", r.FormValue("machine"),
			)
			return
		}
		if i < 0 || i >= len(machines) {
			http.Error(w, "Machine id out of range", http.StatusInternalServerError)
			logger.Error(
				"Machine ID out of range",
				"machine_id", i,
				"max_value", len(machines),
			)
			return
		}

		mach := machines[i]
		for _, port := range mach.Ports {
			l := logger.With(
				"port", port,
				"index", i,
				slog.Group("machine",
					"name", mach.Name,
					"mac", mach.MACAddress,
				),
			)
			err := sendMagicPacket(fmt.Sprintf("%s:%d", broadcastIP, port), mach.MACAddress)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to send magic packet to machine: %v", err), http.StatusInternalServerError)
				l.Error("Failed to send magic packet", "error", err)
				return
			}
			l.Debug("Magic packet sent")
		}

		err = tpl.Execute(w, pageData{Machines: machines, SentName: mach.Name})
		if err != nil {
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			logger.Error("Failed to execute template", "error", err)
			return
		}
	}
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := make([]byte, 6)
		_, _ = rand.Read(out)
		ctx := context.WithValue(r.Context(), requestIDKey, hex.EncodeToString(out))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withRequestFields(log *slog.Logger, r *http.Request) *slog.Logger {
	return log.
		With("request_id", r.Context().Value(requestIDKey)).
		With(slog.Group(
			"request",

			"method", r.Method,
			"path", r.URL.Path,
			"agent", r.UserAgent(),
		))
}
