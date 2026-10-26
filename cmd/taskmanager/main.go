package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/example/taskmanager/internal/config"
	"github.com/example/taskmanager/internal/storage/cache"
	"github.com/example/taskmanager/internal/storage/mysql"
	"github.com/example/taskmanager/internal/usecase"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		panic(err)
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	if err := run(cfg); err != nil {
		panic(err)
	}
}

func run(cfg config.Config) error {
	log, _ := zap.NewProduction()
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("connecting to mysql")
	db, err := mysql.Open(ctx, mysql.ConnectionParams{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Database: cfg.DBName,
	})
	if err != nil {
		return fmt.Errorf("connect to mysql: %w", err)
	}
	defer db.Close()

	log.Info("applying migrations")
	if err := mysql.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate mysql: %w", err)
	}

	redisClient := cache.Connect(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redisClient.Close()

	storage := mysql.New(db)
	taskCache := cache.New(redisClient, cfg.CacheTTL, log)

	u := usecase.New(storage, taskCache, log, cfg.JWTSecret, cfg.JWTTTL)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/register", u.Register)
	mux.HandleFunc("POST /api/v1/login", u.Login)

	mux.HandleFunc("POST /api/v1/teams", u.RequireAuth(u.CreateTeam))
	mux.HandleFunc("GET /api/v1/teams", u.RequireAuth(u.ListTeams))
	mux.HandleFunc("POST /api/v1/teams/{id}/invite", u.RequireAuth(u.Invite))
	mux.HandleFunc("GET /api/v1/teams/{id}/stats", u.RequireAuth(u.TeamStats))

	mux.HandleFunc("POST /api/v1/tasks", u.RequireAuth(u.CreateTask))
	mux.HandleFunc("GET /api/v1/tasks", u.RequireAuth(u.ListTasks))
	mux.HandleFunc("PUT /api/v1/tasks/{id}", u.RequireAuth(u.UpdateTask))
	mux.HandleFunc("GET /api/v1/tasks/{id}/history", u.RequireAuth(u.TaskHistory))
	mux.HandleFunc("POST /api/v1/tasks/{id}/comments", u.RequireAuth(u.AddComment))
	mux.HandleFunc("GET /api/v1/tasks/{id}/comments", u.RequireAuth(u.ListComments))

	mux.HandleFunc("GET /swagger/openapi.yaml", usecase.OpenAPISpec)
	mux.HandleFunc("GET /swagger/{$}", usecase.SwaggerUI)
	mux.Handle("GET /swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           usecase.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownChan := make(chan error, 1)
	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		shutdownChan <- srv.Shutdown(shutdownCtx)
	}()

	log.Info("http server listening", zap.String("addr", cfg.HTTPAddr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return <-shutdownChan
}
