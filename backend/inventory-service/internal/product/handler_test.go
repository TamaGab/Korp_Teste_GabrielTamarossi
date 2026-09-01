package product_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/inventory-service/internal/database"
	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/inventory-service/internal/product"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserCanCreateAndListProduct(t *testing.T) {
	router := newTestRouter(t)

	createResponse := performRequest(router, http.MethodPost, "/products", `{
		"code":"LAP01",
		"description":"Laptop",
		"stock":7
	}`)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("POST /products status = %d, want %d; body = %s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}

	var created product.Product
	decodeResponse(t, createResponse, &created)
	if created.ID == 0 || created.Code != "LAP01" || created.Description != "Laptop" || created.Stock != 7 {
		t.Fatalf("POST /products body = %+v, want persisted Laptop", created)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("POST /products timestamps = %v / %v, want persisted timestamps", created.CreatedAt, created.UpdatedAt)
	}

	listResponse := performRequest(router, http.MethodGet, "/products", "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("GET /products status = %d, want %d; body = %s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}

	var products []product.Product
	decodeResponse(t, listResponse, &products)
	if len(products) != 1 || products[0].ID != created.ID {
		t.Fatalf("GET /products body = %+v, want created Product", products)
	}
}

func TestCreateProductRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid code", body: `{"code":"lap01","description":"Laptop","stock":1}`},
		{name: "blank description", body: `{"code":"LAP01","description":"   ","stock":1}`},
		{name: "negative stock", body: `{"code":"LAP01","description":"Laptop","stock":-1}`},
		{name: "fractional stock", body: `{"code":"LAP01","description":"Laptop","stock":1.5}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(newTestRouter(t), http.MethodPost, "/products", test.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("POST /products status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
			}

			var body map[string]string
			decodeResponse(t, response, &body)
			if body["error"] == "" || len(body) != 1 {
				t.Fatalf("POST /products error = %#v, want {error: message}", body)
			}
		})
	}
}

func TestCreateProductRejectsDuplicateCode(t *testing.T) {
	router := newTestRouter(t)
	body := `{"code":"LAP01","description":"Laptop","stock":1}`
	if response := performRequest(router, http.MethodPost, "/products", body); response.Code != http.StatusCreated {
		t.Fatalf("first POST /products status = %d, want %d", response.Code, http.StatusCreated)
	}

	response := performRequest(router, http.MethodPost, "/products", body)
	if response.Code != http.StatusConflict {
		t.Fatalf("duplicate POST /products status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}

	var errorBody map[string]string
	decodeResponse(t, response, &errorBody)
	if errorBody["error"] != "product code already exists" {
		t.Fatalf("duplicate POST /products error = %q, want product code already exists", errorBody["error"])
	}
}

func TestConcurrentCreateProductAllowsOnlyOneProductCode(t *testing.T) {
	router := newTestRouter(t)
	body := `{"code":"LAP01","description":"Laptop","stock":1}`

	responses := make(chan *httptest.ResponseRecorder, 2)
	start := make(chan struct{})
	var requests sync.WaitGroup
	for range 2 {
		requests.Add(1)
		go func() {
			defer requests.Done()
			<-start
			responses <- performRequest(router, http.MethodPost, "/products", body)
		}()
	}

	close(start)
	requests.Wait()
	close(responses)

	statusCounts := map[int]int{}
	for response := range responses {
		statusCounts[response.Code]++
	}
	if statusCounts[http.StatusCreated] != 1 || statusCounts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent POST /products statuses = %#v, want one 201 and one 409", statusCounts)
	}
}

func TestDatabasePreservesProductInvariants(t *testing.T) {
	pool := newTestPool(t)
	tests := []struct {
		name        string
		code        string
		description string
		stock       int
	}{
		{name: "invalid code", code: "lap01", description: "Laptop", stock: 1},
		{name: "blank description", code: "LAP01", description: "   ", stock: 1},
		{name: "whitespace description", code: "LAP01", description: "\t\n", stock: 1},
		{name: "negative stock", code: "LAP01", description: "Laptop", stock: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := pool.Exec(context.Background(), `
				INSERT INTO products (code, description, stock) VALUES ($1, $2, $3)
			`, test.code, test.description, test.stock)
			if err == nil {
				t.Fatal("database accepted an invalid Product")
			}
		})
	}
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	pool := newTestPool(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	product.RegisterRoutes(router, pool)
	return router
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

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

	if err := database.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "TRUNCATE products RESTART IDENTITY"); err != nil {
		t.Fatalf("reset products: %v", err)
	}
	return pool
}

func performRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
}
