package handlers

import (
	"github.com/go-chi/chi"
	_ "github.com/gpng/order-bot/docs" // required for generating docs
)

// Routes for app
func (s *Handlers) Routes() chi.Router {
	router := chi.NewRouter()

	router.Get("/", s.handleStatus())

	router.Post("/updates", s.handleUpdates())

	return router
}
