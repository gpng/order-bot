// @title Golang API
// @version 0.0.1
// @description Simple REST API using golang

// @contact.name Developers
// @contact.email dev@localhost

// @host localhost:4000
// @BasePath /
package main

import (
	"log"
	"net/http"

	"github.com/gpng/order-bot/services/telegram"

	"github.com/gpng/order-bot/cmd/api/config"
	"github.com/gpng/order-bot/cmd/api/handlers"
	"github.com/gpng/order-bot/services/logger"
	"github.com/gpng/order-bot/services/postgres"
	"github.com/gpng/order-bot/sqlc/models"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("failed to load env vars: %v", err)
	}

	// initialise services
	l := logger.New()
	defer l.Sync()

	db, err := postgres.New(cfg.DbHost, cfg.DbUser, cfg.DbName, cfg.DbPassword)
	if err != nil {
		log.Fatalf("failed to initialise DB connection: %v", err)
	}

	repo := models.New(db)

	bot, err := telegram.New(cfg.BotToken)
	if err != nil {
		log.Fatalf("failed to initialise bot: %v", err)
	}

	handlers := handlers.New(l, db, repo, bot)

	// initialise main router with basic middlewares, cors settings etc
	router := mainRouter()

	// mount services
	router.Mount("/", handlers.Routes())

	err = http.ListenAndServe(":4000", router)
	if err != nil {
		log.Print(err)
	}
}

func mainRouter() chi.Router {
	router := chi.NewRouter()

	// A good base middleware stack
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// stop crawlers
	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("User-agent: *\nDisallow: /"))
	})

	return router
}
