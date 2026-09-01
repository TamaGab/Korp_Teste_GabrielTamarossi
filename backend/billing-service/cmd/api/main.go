package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/billing-service/internal/database"
	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/billing-service/internal/health"
	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/billing-service/internal/invoice"
	"github.com/gin-gonic/gin"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("billing service stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	databaseURL, err := requiredEnvironment("DATABASE_URL")
	if err != nil {
		return err
	}
	port, err := requiredEnvironment("PORT")
	if err != nil {
		return err
	}
	allowedOrigin, err := requiredEnvironment("CORS_ALLOWED_ORIGIN")
	if err != nil {
		return err
	}
	inventoryServiceURL, err := requiredEnvironment("INVENTORY_SERVICE_URL")
	if err != nil {
		return err
	}

	logger.Info("starting billing service")

	connectionContext, cancelConnection := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelConnection()

	pool, err := database.NewPool(connectionContext, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to billing database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connection established")
	if err := database.Migrate(connectionContext, pool); err != nil {
		return fmt.Errorf("migrate billing database: %w", err)
	}
	logger.Info("database migrations applied")

	router := gin.New()
	router.Use(gin.Recovery(), corsMiddleware(allowedOrigin))
	router.GET("/health", health.Handler)
	invoice.RegisterRoutes(router, pool, inventoryServiceURL, &http.Client{Timeout: 5 * time.Second})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", "port", port)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-shutdownSignal.Done():
		logger.Info("shutdown signal received")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shut down HTTP server: %w", err)
	}

	logger.Info("billing service stopped gracefully")
	return nil
}

func requiredEnvironment(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is required", name)
	}
	return value, nil
}

func corsMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestOrigin := c.GetHeader("Origin")
		if originAllowed(requestOrigin, allowedOrigin) {
			c.Header("Access-Control-Allow-Origin", requestOrigin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func originAllowed(requestOrigin, configuredOrigin string) bool {
	if requestOrigin == configuredOrigin {
		return true
	}

	requestURL, requestErr := url.Parse(requestOrigin)
	configuredURL, configuredErr := url.Parse(configuredOrigin)
	if requestErr != nil || configuredErr != nil || !plainOrigin(requestURL) || !plainOrigin(configuredURL) {
		return false
	}
	if requestURL.Scheme != configuredURL.Scheme || requestURL.Port() != configuredURL.Port() {
		return false
	}

	return localHostname(requestURL.Hostname()) && localHostname(configuredURL.Hostname())
}

func plainOrigin(origin *url.URL) bool {
	return origin.Scheme != "" && origin.Host != "" && origin.User == nil && origin.Path == "" &&
		origin.RawQuery == "" && origin.Fragment == ""
}

func localHostname(hostname string) bool {
	return hostname == "localhost" || hostname == "127.0.0.1"
}
