package invoice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/billing-service/internal/database"
	"github.com/TamaGab/Korp_Teste_GabrielTamarossi/backend/billing-service/internal/invoice"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserCanCreateOpenInvoiceWithProductSnapshots(t *testing.T) {
	inventory := newInventoryServer(t, map[string]string{
		"/products": `[{"id":1,"code":"LAP01","description":"Laptop","stock":7},{"id":2,"code":"MON01","description":"Monitor","stock":0}]`,
	})
	router := newTestRouter(t, inventory.URL)

	response := performRequest(router, http.MethodPost, "/invoices", `{
		"lines":[
			{"productCode":"LAP01","quantity":2},
			{"productCode":"MON01","quantity":3}
		]
	}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("POST /invoices status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var created invoice.Invoice
	decodeResponse(t, response, &created)
	if created.Number != "0001" || created.Status != "OPEN" {
		t.Fatalf("POST /invoices identity = %q / %q, want 0001 / OPEN", created.Number, created.Status)
	}
	if len(created.Lines) != 2 {
		t.Fatalf("POST /invoices lines = %+v, want two lines", created.Lines)
	}
	if created.Lines[0].Code != "LAP01" || created.Lines[0].Description != "Laptop" || created.Lines[0].Quantity != 2 {
		t.Fatalf("POST /invoices first line = %+v, want Laptop snapshot with quantity 2", created.Lines[0])
	}
	if created.Lines[1].Code != "MON01" || created.Lines[1].Description != "Monitor" || created.Lines[1].Quantity != 3 {
		t.Fatalf("POST /invoices second line = %+v, want Monitor snapshot with quantity 3", created.Lines[1])
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("POST /invoices createdAt is zero, want persisted timestamp")
	}
}

func TestCreatedInvoicesReceiveIncreasingNumbers(t *testing.T) {
	inventory := newInventoryServer(t, map[string]string{
		"/products": `[{"id":1,"code":"LAP01","description":"Laptop","stock":0}]`,
	})
	router := newTestRouter(t, inventory.URL)
	body := `{"lines":[{"productCode":"LAP01","quantity":20}]}`

	firstResponse := performRequest(router, http.MethodPost, "/invoices", body)
	secondResponse := performRequest(router, http.MethodPost, "/invoices", body)
	if firstResponse.Code != http.StatusCreated || secondResponse.Code != http.StatusCreated {
		t.Fatalf("POST /invoices statuses = %d / %d, want 201 / 201", firstResponse.Code, secondResponse.Code)
	}

	var first invoice.Invoice
	var second invoice.Invoice
	decodeResponse(t, firstResponse, &first)
	decodeResponse(t, secondResponse, &second)
	if first.Number != "0001" || second.Number != "0002" {
		t.Fatalf("POST /invoices numbers = %q / %q, want 0001 / 0002", first.Number, second.Number)
	}
}

func TestCreateInvoiceRejectsInvalidInput(t *testing.T) {
	inventory := newInventoryServer(t, map[string]string{
		"/products": `[{"id":1,"code":"LAP01","description":"Laptop","stock":7}]`,
	})
	tests := []struct {
		name string
		body string
	}{
		{name: "missing lines", body: `{}`},
		{name: "empty lines", body: `{"lines":[]}`},
		{name: "invalid product", body: `{"lines":[{"productCode":"invalid","quantity":1}]}`},
		{name: "zero quantity", body: `{"lines":[{"productCode":"LAP01","quantity":0}]}`},
		{name: "negative quantity", body: `{"lines":[{"productCode":"LAP01","quantity":-1}]}`},
		{name: "fractional quantity", body: `{"lines":[{"productCode":"LAP01","quantity":1.5}]}`},
		{name: "duplicate product", body: `{"lines":[{"productCode":"LAP01","quantity":1},{"productCode":"LAP01","quantity":2}]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRequest(newTestRouter(t, inventory.URL), http.MethodPost, "/invoices", test.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("POST /invoices status = %d, want 400; body = %s", response.Code, response.Body.String())
			}
			var body map[string]any
			decodeResponse(t, response, &body)
			if body["error"] == "" {
				t.Fatalf("POST /invoices error = %#v, want a message", body["error"])
			}
		})
	}
}

func TestCreateInvoiceRejectsMissingProductWithoutUsingANumber(t *testing.T) {
	inventory := newInventoryServer(t, map[string]string{
		"/products": `[{"id":1,"code":"LAP01","description":"Laptop","stock":7}]`,
	})
	router := newTestRouter(t, inventory.URL)

	notFoundResponse := performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"XXX99","quantity":1}]}`)
	if notFoundResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST /invoices missing Product status = %d, want 422; body = %s", notFoundResponse.Code, notFoundResponse.Body.String())
	}
	var errorBody map[string]any
	decodeResponse(t, notFoundResponse, &errorBody)
	if errorBody["error"] != "product not found" || errorBody["productCode"] != "XXX99" {
		t.Fatalf("POST /invoices missing Product body = %#v, want Product XXX99 error", errorBody)
	}

	createdResponse := performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":1}]}`)
	var created invoice.Invoice
	decodeResponse(t, createdResponse, &created)
	if createdResponse.Code != http.StatusCreated || created.Number != "0001" {
		t.Fatalf("POST /invoices after rejection = %d / %q, want 201 / 0001", createdResponse.Code, created.Number)
	}
}

func TestCreateInvoiceReturnsBadGatewayWhenInventoryFails(t *testing.T) {
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, `{"error":"unavailable"}`, http.StatusServiceUnavailable)
	}))
	t.Cleanup(inventory.Close)

	response := performRequest(newTestRouter(t, inventory.URL), http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":1}]}`)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("POST /invoices inventory failure status = %d, want 502; body = %s", response.Code, response.Body.String())
	}
	var body map[string]string
	decodeResponse(t, response, &body)
	if body["error"] != "could not validate products" {
		t.Fatalf("POST /invoices inventory failure error = %q, want could not validate products", body["error"])
	}
}

func newInventoryServer(t *testing.T, products map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(response, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		product, exists := products[request.URL.Path]
		if !exists {
			http.Error(response, `{"error":"product not found"}`, http.StatusNotFound)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(product))
	}))
	t.Cleanup(server.Close)
	return server
}

func newTestRouter(t *testing.T, inventoryURL string) http.Handler {
	t.Helper()
	pool := newTestPool(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoice.RegisterRoutes(router, pool, inventoryURL, http.DefaultClient)
	return router
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		databaseURL = "postgres://billing:billing_dev_password@localhost:5434/billing?sslmode=disable"
	}

	pool, err := database.NewPool(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := database.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "TRUNCATE invoice_lines, invoices RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("reset invoices: %v", err)
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
