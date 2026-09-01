package product_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func TestUserCanGetProductByID(t *testing.T) {
	router := newTestRouter(t)
	createResponse := performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop","stock":7}`)
	var created product.Product
	decodeResponse(t, createResponse, &created)

	response := performRequest(router, http.MethodGet, fmt.Sprintf("/products/%d", created.ID), "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /products/:id status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var found product.Product
	decodeResponse(t, response, &found)
	if found != created {
		t.Fatalf("GET /products/:id body = %+v, want %+v", found, created)
	}
}

func TestGetProductReturnsNotFound(t *testing.T) {
	response := performRequest(newTestRouter(t), http.MethodGet, "/products/999", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("GET /products/:id status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}

	var body map[string]string
	decodeResponse(t, response, &body)
	if body["error"] != "product not found" || len(body) != 1 {
		t.Fatalf("GET /products/:id error = %#v, want {error: product not found}", body)
	}
}

func TestUserCanUpdateProductWithoutChangingItsIdentity(t *testing.T) {
	router := newTestRouter(t)
	createResponse := performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop","stock":7}`)
	var created product.Product
	decodeResponse(t, createResponse, &created)

	response := performRequest(router, http.MethodPut, fmt.Sprintf("/products/%d", created.ID), `{"code":"NOT02","description":"Notebook","stock":4}`)
	if response.Code != http.StatusOK {
		t.Fatalf("PUT /products/:id status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var updated product.Product
	decodeResponse(t, response, &updated)
	if updated.ID != created.ID || updated.CreatedAt != created.CreatedAt {
		t.Fatalf("PUT /products/:id identity = %d / %v, want %d / %v", updated.ID, updated.CreatedAt, created.ID, created.CreatedAt)
	}
	if updated.Code != "NOT02" || updated.Description != "Notebook" || updated.Stock != 4 {
		t.Fatalf("PUT /products/:id body = %+v, want updated Product", updated)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("PUT /products/:id updatedAt = %v, want after %v", updated.UpdatedAt, created.UpdatedAt)
	}

	getResponse := performRequest(router, http.MethodGet, fmt.Sprintf("/products/%d", created.ID), "")
	var persisted product.Product
	decodeResponse(t, getResponse, &persisted)
	if persisted != updated {
		t.Fatalf("GET after PUT body = %+v, want %+v", persisted, updated)
	}

	secondResponse := performRequest(router, http.MethodPut, fmt.Sprintf("/products/%d", created.ID), `{"code":"NOT02","description":"Notebook","stock":4}`)
	var updatedAgain product.Product
	decodeResponse(t, secondResponse, &updatedAgain)
	if !updatedAgain.UpdatedAt.After(updated.UpdatedAt) {
		t.Fatalf("second PUT updatedAt = %v, want after %v", updatedAgain.UpdatedAt, updated.UpdatedAt)
	}
}

func TestUpdateProductRejectsInvalidInput(t *testing.T) {
	router := newTestRouter(t)
	createResponse := performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop","stock":7}`)
	var created product.Product
	decodeResponse(t, createResponse, &created)

	tests := []string{
		`{"code":"lap01","description":"Laptop","stock":1}`,
		`{"code":"LAP01","description":"   ","stock":1}`,
		`{"code":"LAP01","description":"Laptop","stock":-1}`,
		`{"code":"LAP01","description":"Laptop","stock":1.5}`,
	}
	for _, body := range tests {
		response := performRequest(router, http.MethodPut, fmt.Sprintf("/products/%d", created.ID), body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("PUT /products/:id status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
		}
	}
}

func TestUpdateProductReturnsNotFound(t *testing.T) {
	response := performRequest(newTestRouter(t), http.MethodPut, "/products/999", `{"code":"LAP01","description":"Laptop","stock":7}`)
	if response.Code != http.StatusNotFound {
		t.Fatalf("PUT /products/:id status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func TestUpdateProductRejectsDuplicateCode(t *testing.T) {
	router := newTestRouter(t)
	performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop","stock":7}`)
	createResponse := performRequest(router, http.MethodPost, "/products", `{"code":"MON01","description":"Monitor","stock":3}`)
	var monitor product.Product
	decodeResponse(t, createResponse, &monitor)

	response := performRequest(router, http.MethodPut, fmt.Sprintf("/products/%d", monitor.ID), `{"code":"LAP01","description":"Monitor","stock":3}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("PUT /products/:id status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestUserCanDeleteProductAndReuseItsCode(t *testing.T) {
	router := newTestRouter(t)
	createResponse := performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop","stock":7}`)
	var created product.Product
	decodeResponse(t, createResponse, &created)

	deleteResponse := performRequest(router, http.MethodDelete, fmt.Sprintf("/products/%d", created.ID), "")
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("DELETE /products/:id status = %d, want %d; body = %s", deleteResponse.Code, http.StatusNoContent, deleteResponse.Body.String())
	}
	if deleteResponse.Body.Len() != 0 {
		t.Fatalf("DELETE /products/:id body = %q, want empty body", deleteResponse.Body.String())
	}

	getResponse := performRequest(router, http.MethodGet, fmt.Sprintf("/products/%d", created.ID), "")
	if getResponse.Code != http.StatusNotFound {
		t.Fatalf("GET deleted Product status = %d, want %d", getResponse.Code, http.StatusNotFound)
	}

	recreateResponse := performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop novo","stock":2}`)
	if recreateResponse.Code != http.StatusCreated {
		t.Fatalf("POST with deleted Product Code status = %d, want %d; body = %s", recreateResponse.Code, http.StatusCreated, recreateResponse.Body.String())
	}
}

func TestDeleteProductReturnsNotFound(t *testing.T) {
	response := performRequest(newTestRouter(t), http.MethodDelete, "/products/999", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("DELETE /products/:id status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}

	var body map[string]string
	decodeResponse(t, response, &body)
	if body["error"] != "product not found" || len(body) != 1 {
		t.Fatalf("DELETE /products/:id error = %#v, want {error: product not found}", body)
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

func TestStockValidationReturnsEveryBlockingProductWithoutChangingStock(t *testing.T) {
	router := newTestRouter(t)
	firstResponse := performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop","stock":2}`)
	secondResponse := performRequest(router, http.MethodPost, "/products", `{"code":"MON01","description":"Monitor","stock":1}`)
	var laptop product.Product
	var monitor product.Product
	decodeResponse(t, firstResponse, &laptop)
	decodeResponse(t, secondResponse, &monitor)

	response := performRequest(router, http.MethodPost, "/stock/validate", fmt.Sprintf(`{
		"lines":[
			{"inventoryProductId":%d,"quantity":3},
			{"inventoryProductId":9999,"quantity":1},
			{"inventoryProductId":%d,"quantity":2}
		]
	}`, laptop.ID, monitor.ID))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST /stock/validate status = %d, want 422; body = %s", response.Code, response.Body.String())
	}

	var body struct {
		Problems []struct {
			InventoryProductID int    `json:"inventoryProductId"`
			Reason             string `json:"reason"`
			AvailableStock     *int   `json:"availableStock"`
			RequestedAmount    int    `json:"requestedQuantity"`
		} `json:"problems"`
	}
	decodeResponse(t, response, &body)
	if len(body.Problems) != 3 {
		t.Fatalf("POST /stock/validate problems = %+v, want all three blocking Products", body.Problems)
	}
	if body.Problems[0].InventoryProductID != laptop.ID || body.Problems[0].Reason != "insufficient_stock" || body.Problems[0].AvailableStock == nil || *body.Problems[0].AvailableStock != 2 || body.Problems[0].RequestedAmount != 3 {
		t.Fatalf("first validation problem = %+v, want Laptop stock details", body.Problems[0])
	}
	if body.Problems[1].InventoryProductID != 9999 || body.Problems[1].Reason != "product_not_found" || body.Problems[1].AvailableStock != nil {
		t.Fatalf("second validation problem = %+v, want missing Product", body.Problems[1])
	}

	for _, current := range []product.Product{laptop, monitor} {
		getResponse := performRequest(router, http.MethodGet, fmt.Sprintf("/products/%d", current.ID), "")
		var afterValidation product.Product
		decodeResponse(t, getResponse, &afterValidation)
		if afterValidation.Stock != current.Stock {
			t.Fatalf("Product %d stock after validation = %d, want unchanged %d", current.ID, afterValidation.Stock, current.Stock)
		}
	}
}

func TestStockValidationAcceptsEveryAvailableProduct(t *testing.T) {
	router := newTestRouter(t)
	createResponse := performRequest(router, http.MethodPost, "/products", `{"code":"LAP01","description":"Laptop","stock":2}`)
	var laptop product.Product
	decodeResponse(t, createResponse, &laptop)

	response := performRequest(router, http.MethodPost, "/stock/validate", fmt.Sprintf(`{"lines":[{"inventoryProductId":%d,"quantity":2}]}`, laptop.ID))
	if response.Code != http.StatusOK || response.Body.String() != `{"problems":[]}` {
		t.Fatalf("POST /stock/validate = %d / %s, want 200 with no problems", response.Code, response.Body.String())
	}
}

func TestStockValidationRejectsInvalidInput(t *testing.T) {
	for _, body := range []string{
		`{}`,
		`{"lines":[]}`,
		`{"lines":[{"inventoryProductId":0,"quantity":1}]}`,
		`{"lines":[{"inventoryProductId":1,"quantity":0}]}`,
		`{"lines":[{"inventoryProductId":1,"quantity":1},{"inventoryProductId":1,"quantity":2}]}`,
	} {
		response := performRequest(newTestRouter(t), http.MethodPost, "/stock/validate", body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("POST /stock/validate body %s status = %d, want 400", body, response.Code)
		}
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
