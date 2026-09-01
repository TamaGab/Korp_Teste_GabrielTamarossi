package health_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/inventory-service/internal/database"
	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/inventory-service/internal/health"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHealthReportsServiceAndDatabaseAvailable(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		databaseURL = "postgres://inventory:inventory_dev_password@localhost:5433/inventory?sslmode=disable"
	}
	pool, err := database.NewPool(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", health.Handler(pool))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", response.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	want := map[string]string{"status": "ok", "service": "ok", "database": "ok"}
	for key, value := range want {
		if body[key] != value {
			t.Errorf("GET /health %s = %q, want %q", key, body[key], value)
		}
	}
}

func TestHealthReportsDatabaseUnavailableWhileServiceResponds(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://unused:unused@localhost:1/unused")
	if err != nil {
		t.Fatalf("create unavailable database pool: %v", err)
	}
	pool.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", health.Handler(pool))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /health status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	want := map[string]string{"status": "degraded", "service": "ok", "database": "unavailable"}
	for key, value := range want {
		if body[key] != value {
			t.Errorf("GET /health %s = %q, want %q", key, body[key], value)
		}
	}
}
