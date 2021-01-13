package handlers

import (
	"github.com/go-chi/chi"
)

// Routes for app
func (h *Handlers) Routes() chi.Router {
	router := chi.NewRouter()

	router.Get("/", h.handleStatus())

	router.Route("/" + h.BotToken, func (r chi.Router) {
		r.Post("/", h.handleUpdates())
	})

	return router
}
