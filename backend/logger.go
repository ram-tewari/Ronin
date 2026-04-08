package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

var logger *slog.Logger

// InitLogger initializes the global structured logger that writes to ronin.log
func InitLogger() error {
	logFile, err := os.OpenFile("ronin.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger = slog.New(handler)
	logger.Info("Logger initialized", slog.String("file", "ronin.log"))

	return nil
}

// TelemetryRequest represents a log entry from the frontend
type TelemetryRequest struct {
	Level   string                 `json:"level"`
	Source  string                 `json:"source"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

// logHandler handles POST /log — receives frontend telemetry and writes to slog
func logHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to parse telemetry request",
			slog.String("error", err.Error()),
			slog.String("source", "backend"))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Convert details map to slog attributes
	attrs := []slog.Attr{
		slog.String("source", req.Source),
		slog.String("message", req.Message),
	}

	for key, value := range req.Details {
		attrs = append(attrs, slog.Any(key, value))
	}

	// Log at the appropriate level
	switch req.Level {
	case "info":
		logger.LogAttrs(r.Context(), slog.LevelInfo, "Frontend telemetry", attrs...)
	case "warn":
		logger.LogAttrs(r.Context(), slog.LevelWarn, "Frontend telemetry", attrs...)
	case "error":
		logger.LogAttrs(r.Context(), slog.LevelError, "Frontend telemetry", attrs...)
	default:
		logger.LogAttrs(r.Context(), slog.LevelInfo, "Frontend telemetry", attrs...)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
