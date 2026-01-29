package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-microservice/handlers"
	"go-microservice/metrics"
	"go-microservice/services"
	"go-microservice/utils"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

func main() {
	logger := utils.NewLogger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	audit := utils.NewAuditLogger(logger, 10000)
	errs := utils.NewErrorReporter(logger, 1000)
	notifier := services.NewNotifier(logger, 10000)

	audit.Start(ctx)
	errs.Start(ctx)
	notifier.Start(ctx)

	userSvc := services.NewInMemoryUserService()

	userHandler := handlers.NewUserHandler(userSvc, audit, notifier, errs)

	r := mux.NewRouter()

	// Middleware (порядок важен: метрики вокруг всего, дальше rate limit)
	r.Use(metrics.Middleware())
	limiter := rate.NewLimiter(rate.Limit(1000), 5000)
	r.Use(utils.RateLimitMiddleware(limiter))

	// Routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/users", userHandler.ListUsers).Methods(http.MethodGet).Name("list_users")
	api.HandleFunc("/users/{id}", userHandler.GetUser).Methods(http.MethodGet).Name("get_user")
	api.HandleFunc("/users", userHandler.CreateUser).Methods(http.MethodPost).Name("create_user")
	api.HandleFunc("/users/{id}", userHandler.UpdateUser).Methods(http.MethodPut).Name("update_user")
	api.HandleFunc("/users/{id}", userHandler.DeleteUser).Methods(http.MethodDelete).Name("delete_user")

	// Service endpoints (обычно /metrics не стоит ограничивать rate limit’ом, но ТЗ это не запрещает)
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet).Name("metrics")
	r.HandleFunc("/healthz", handlers.Healthz).Methods(http.MethodGet).Name("healthz")

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	log.Println("shutdown complete")
}

