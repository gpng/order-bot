package handlers

import (
	"net/http"
)

type statusResponse struct {
	Version int `json:"version"`
}

// handleStatus returns the current api version
func (h *Handlers) handleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := statusResponse{
			Version: 1,
		}
		respond(w, dataMessage(status, "API responding"))
	}
}
