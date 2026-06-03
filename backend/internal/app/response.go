package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
}

func ok(w http.ResponseWriter, data interface{}, status int) {
	writeJSON(w, status, envelope{Success: true, Data: data})
}

func okMessage(w http.ResponseWriter, message string, status int) {
	writeJSON(w, status, envelope{Success: true, Message: message})
}

func fail(w http.ResponseWriter, status int, code, message string) {
	slog.Default().Warn("request failed", "status", status, "code", code, "message", message)
	writeJSON(w, status, envelope{Success: false, Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, payload envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
