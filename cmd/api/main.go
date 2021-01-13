package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gpng/order-bot/services/telegram"

	"github.com/gpng/order-bot/cmd/api/config"
	"github.com/gpng/order-bot/cmd/api/handlers"
	"github.com/gpng/order-bot/services/logger"
	"github.com/gpng/order-bot/services/postgres"
	"github.com/gpng/order-bot/sqlc/models"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("failed to load env vars: %v", err)
	}
	log.Printf("cfg: %v\n", cfg)

	// initialise services
	l := logger.New()
	defer l.Sync()

	db, err := postgres.New(cfg.DbHost, cfg.DbUser, cfg.DbName, cfg.DbPassword)
	if err != nil {
		log.Fatalf("failed to initialise DB connection: %v", err)
	}

	repo := models.New(db)

	redisPool := &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(cfg.RedisURL, redis.DialPassword(cfg.RedisPassword))
		},
	}

	conn := redisPool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("redis connection successful")

	bot, err := telegram.New(cfg.BotToken)
	if err != nil {
		log.Fatalf("failed to initialise bot: %v", err)
	}

	enqeuer := work.NewEnqueuer(cfg.RedisNamespace, redisPool)
	pool := work.NewWorkerPool(handlers.Handlers{}, 10, cfg.RedisNamespace, redisPool)

	pool.Middleware(func(c *handlers.Handlers, job *work.Job, next work.NextMiddlewareFunc) error {
		c.Queue = enqeuer
		c.Logger = l
		c.DB = db
		c.Repo = repo
		c.Bot = bot
		return next()
	})

	pool.JobWithOptions(string(handlers.JobNotifyExpiry), work.JobOptions{
		MaxConcurrency: 1,
		MaxFails:       3,
	}, (*handlers.Handlers).JobNotifyExpiry)

	h := handlers.New(cfg.BotToken, l, db, repo, bot, enqeuer)

	// initialise main router with basic middlewares, cors settings etc
	router := mainRouter()

	router.Mount("/", h.Routes())

	log.Println("starting worker pool...")
	pool.Start()
	// defer pool.Stop()

	if err != nil {
		log.Printf("err: %v\n", err)
	}

	log.Println("listening to port " + cfg.Port)
	err = http.ListenAndServe(":"+cfg.Port, router)
	if err != nil {
		log.Print(err)
	}

	// Wait for a signal to quit:
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	<-signalChan
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
