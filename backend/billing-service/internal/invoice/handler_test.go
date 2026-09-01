package invoice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
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

func TestUserCanListInvoices(t *testing.T) {
	inventory := newInventoryServer(t, map[string]string{
		"/products": `[{"id":1,"code":"LAP01","description":"Laptop","stock":7}]`,
	})
	router := newTestRouter(t, inventory.URL)

	createdResponse := performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":2}]}`)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("POST /invoices status = %d, want 201; body = %s", createdResponse.Code, createdResponse.Body.String())
	}

	response := performRequest(router, http.MethodGet, "/invoices", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /invoices status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var invoices []invoice.Invoice
	decodeResponse(t, response, &invoices)
	if len(invoices) != 1 {
		t.Fatalf("GET /invoices body = %+v, want one Invoice", invoices)
	}
	if invoices[0].Number != "0001" || invoices[0].Status != "OPEN" || invoices[0].CreatedAt.IsZero() {
		t.Fatalf("GET /invoices Invoice = %+v, want formatted number, OPEN status and timestamp", invoices[0])
	}
}

func TestListingInvoicesReturnsAnEmptyCollection(t *testing.T) {
	response := performRequest(newTestRouter(t, "http://inventory.invalid"), http.MethodGet, "/invoices", "")

	if response.Code != http.StatusOK || response.Body.String() != "[]" {
		t.Fatalf("GET /invoices = %d / %s, want 200 / []", response.Code, response.Body.String())
	}
}

func TestUserCanConsultInvoiceWithHistoricalProductData(t *testing.T) {
	products := `[{"id":1,"code":"LAP01","description":"Laptop original","stock":7}]`
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(products))
	}))
	t.Cleanup(inventory.Close)
	router := newTestRouter(t, inventory.URL)

	createdResponse := performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":2}]}`)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("POST /invoices status = %d, want 201; body = %s", createdResponse.Code, createdResponse.Body.String())
	}
	products = `[{"id":1,"code":"NEW99","description":"Descrição alterada","stock":7}]`

	response := performRequest(router, http.MethodGet, "/invoices/0001", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /invoices/0001 status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var found invoice.Invoice
	decodeResponse(t, response, &found)
	if found.Number != "0001" || found.Status != "OPEN" {
		t.Fatalf("GET /invoices/0001 identity = %q / %q, want 0001 / OPEN", found.Number, found.Status)
	}
	if len(found.Lines) != 1 {
		t.Fatalf("GET /invoices/0001 lines = %+v, want one historical Invoice Line", found.Lines)
	}
	if found.Lines[0].Code != "LAP01" || found.Lines[0].Description != "Laptop original" || found.Lines[0].Quantity != 2 {
		t.Fatalf("GET /invoices/0001 line = %+v, want original Product snapshot", found.Lines[0])
	}
}

func TestConsultingMissingInvoiceReturnsNotFound(t *testing.T) {
	router := newTestRouter(t, "http://inventory.invalid")

	for _, path := range []string{"/invoices/9999", "/invoices/invalid"} {
		response := performRequest(router, http.MethodGet, path, "")
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want 404; body = %s", path, response.Code, response.Body.String())
		}
		var body map[string]string
		decodeResponse(t, response, &body)
		if body["error"] != "invoice not found" {
			t.Fatalf("GET %s error = %q, want invoice not found", path, body["error"])
		}
	}
}

func TestOpenAndClosedInvoicesRemainConsultable(t *testing.T) {
	inventory := newInventoryServer(t, map[string]string{
		"/products": `[{"id":1,"code":"LAP01","description":"Laptop","stock":7}]`,
	})
	router, pool := newTestRouterWithPool(t, inventory.URL)
	body := `{"lines":[{"productCode":"LAP01","quantity":1}]}`
	performRequest(router, http.MethodPost, "/invoices", body)
	performRequest(router, http.MethodPost, "/invoices", body)
	if _, err := pool.Exec(context.Background(), "UPDATE invoices SET status = 'CLOSED' WHERE number = 1"); err != nil {
		t.Fatalf("close Invoice fixture: %v", err)
	}

	listResponse := performRequest(router, http.MethodGet, "/invoices", "")
	var invoices []invoice.Invoice
	decodeResponse(t, listResponse, &invoices)
	if listResponse.Code != http.StatusOK || len(invoices) != 2 {
		t.Fatalf("GET /invoices = %d / %+v, want both Invoices", listResponse.Code, invoices)
	}
	statuses := map[string]string{invoices[0].Number: invoices[0].Status, invoices[1].Number: invoices[1].Status}
	if statuses["0001"] != "CLOSED" || statuses["0002"] != "OPEN" {
		t.Fatalf("GET /invoices statuses = %+v, want 0001 CLOSED and 0002 OPEN", statuses)
	}

	detailResponse := performRequest(router, http.MethodGet, "/invoices/0001", "")
	var closed invoice.Invoice
	decodeResponse(t, detailResponse, &closed)
	if detailResponse.Code != http.StatusOK || closed.Status != "CLOSED" {
		t.Fatalf("GET /invoices/0001 = %d / %+v, want consultable CLOSED Invoice", detailResponse.Code, closed)
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

func TestUserCanPreparePrintableHTMLForAnEligibleInvoice(t *testing.T) {
	var validationBody map[string]any
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/products" {
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`[{"id":1,"code":"LAP01","description":"<Laptop> & Cia","stock":2}]`))
			return
		}
		if request.URL.Path != "/stock/validate" || request.Method != http.MethodPost {
			http.NotFound(response, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&validationBody); err != nil {
			t.Errorf("decode inventory validation request: %v", err)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"problems":[]}`))
	}))
	t.Cleanup(inventory.Close)
	router := newTestRouter(t, inventory.URL)
	performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":2}]}`)

	response := performRequest(router, http.MethodPost, "/invoices/0001/prepare-print", "")
	if response.Code != http.StatusOK {
		t.Fatalf("POST /invoices/0001/prepare-print status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
	var prepared struct {
		HTML string `json:"html"`
	}
	decodeResponse(t, response, &prepared)
	for _, expected := range []string{"Nota Fiscal", "0001", "LAP01", "&lt;Laptop&gt; &amp; Cia", "2"} {
		if !strings.Contains(prepared.HTML, expected) {
			t.Fatalf("print HTML %q does not contain %q", prepared.HTML, expected)
		}
	}
	for _, forbidden := range []string{"OPEN", "Aberta", "createdAt", "Cliente", "Data", "Valor", "Imposto", "Total", "R$"} {
		if strings.Contains(prepared.HTML, forbidden) {
			t.Fatalf("print HTML contains forbidden content %q: %s", forbidden, prepared.HTML)
		}
	}
	lines := validationBody["lines"].([]any)
	line := lines[0].(map[string]any)
	if line["inventoryProductId"] != float64(1) || line["quantity"] != float64(2) {
		t.Fatalf("inventory validation body = %#v, want persisted Product identity and quantity", validationBody)
	}
}

func TestPreparingPrintReturnsEveryStockProblemWithProductCodes(t *testing.T) {
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/products" {
			_, _ = response.Write([]byte(`[{"id":1,"code":"LAP01","description":"Laptop"},{"id":2,"code":"MON01","description":"Monitor"}]`))
			return
		}
		response.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = response.Write([]byte(`{"problems":[{"inventoryProductId":1,"reason":"insufficient_stock","availableStock":1,"requestedQuantity":2},{"inventoryProductId":2,"reason":"product_not_found","requestedQuantity":1}]}`))
	}))
	t.Cleanup(inventory.Close)
	router := newTestRouter(t, inventory.URL)
	performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":2},{"productCode":"MON01","quantity":1}]}`)

	response := performRequest(router, http.MethodPost, "/invoices/0001/prepare-print", "")
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("POST prepare-print status = %d, want 422; body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Problems []struct {
			ProductCode string `json:"productCode"`
			Reason      string `json:"reason"`
		} `json:"problems"`
	}
	decodeResponse(t, response, &body)
	if len(body.Problems) != 2 || body.Problems[0].ProductCode != "LAP01" || body.Problems[0].Reason != "insufficient_stock" || body.Problems[1].ProductCode != "MON01" || body.Problems[1].Reason != "product_not_found" {
		t.Fatalf("prepare-print problems = %+v, want both Product Codes and reasons", body.Problems)
	}
}

func TestPreparingPrintRejectsClosedAndPendingInvoices(t *testing.T) {
	inventory := newInventoryServer(t, map[string]string{
		"/products":       `[{"id":1,"code":"LAP01","description":"Laptop"}]`,
		"/stock/validate": `{"problems":[]}`,
	})
	router, pool := newTestRouterWithPool(t, inventory.URL)
	body := `{"lines":[{"productCode":"LAP01","quantity":1}]}`
	performRequest(router, http.MethodPost, "/invoices", body)
	performRequest(router, http.MethodPost, "/invoices", body)
	if _, err := pool.Exec(context.Background(), "UPDATE invoices SET status = 'CLOSED' WHERE number = 1"); err != nil {
		t.Fatalf("close Invoice fixture: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE invoices SET closing_pending = TRUE WHERE number = 2"); err != nil {
		t.Fatalf("mark pending Invoice fixture: %v", err)
	}

	for _, number := range []string{"0001", "0002"} {
		response := performRequest(router, http.MethodPost, "/invoices/"+number+"/prepare-print", "")
		if response.Code != http.StatusConflict {
			t.Fatalf("POST prepare-print for Invoice %s status = %d, want 409; body = %s", number, response.Code, response.Body.String())
		}
	}
}

func TestPreparingPrintKeepsOpenInvoiceWhenInventoryIsUnavailable(t *testing.T) {
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/products" {
			_, _ = response.Write([]byte(`[{"id":1,"code":"LAP01","description":"Laptop"}]`))
			return
		}
		http.Error(response, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(inventory.Close)
	router := newTestRouter(t, inventory.URL)
	performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":1}]}`)

	response := performRequest(router, http.MethodPost, "/invoices/0001/prepare-print", "")
	if response.Code != http.StatusBadGateway {
		t.Fatalf("POST prepare-print status = %d, want 502; body = %s", response.Code, response.Body.String())
	}
	getResponse := performRequest(router, http.MethodGet, "/invoices/0001", "")
	var current invoice.Invoice
	decodeResponse(t, getResponse, &current)
	if current.Status != "OPEN" {
		t.Fatalf("Invoice status after unavailable inventory = %q, want OPEN", current.Status)
	}
}

func TestUserCanCloseAnOpenInvoiceAndConsumeItsPersistedLines(t *testing.T) {
	var consumptionBody struct {
		InvoiceNumber string `json:"invoiceNumber"`
		Lines         []struct {
			InventoryProductID int `json:"inventoryProductId"`
			Quantity           int `json:"quantity"`
		} `json:"lines"`
	}
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/products":
			_, _ = response.Write([]byte(`[{"id":1,"code":"LAP01","description":"Laptop","stock":5}]`))
		case "/stock/consume":
			if request.Method != http.MethodPost {
				http.NotFound(response, request)
				return
			}
			if err := json.NewDecoder(request.Body).Decode(&consumptionBody); err != nil {
				t.Errorf("decode Stock Consumption request: %v", err)
			}
			_, _ = response.Write([]byte(`{"problems":[]}`))
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(inventory.Close)
	router := newTestRouter(t, inventory.URL)
	performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":2}]}`)

	closeResponse := performRequest(router, http.MethodPost, "/invoices/0001/close", "")
	if closeResponse.Code != http.StatusOK {
		t.Fatalf("POST /invoices/0001/close status = %d, want 200; body = %s", closeResponse.Code, closeResponse.Body.String())
	}
	if consumptionBody.InvoiceNumber != "0001" || len(consumptionBody.Lines) != 1 || consumptionBody.Lines[0].InventoryProductID != 1 || consumptionBody.Lines[0].Quantity != 2 {
		t.Fatalf("Stock Consumption body = %+v, want Invoice 0001 and persisted Laptop line", consumptionBody)
	}

	getResponse := performRequest(router, http.MethodGet, "/invoices/0001", "")
	var closed invoice.Invoice
	decodeResponse(t, getResponse, &closed)
	if closed.Status != "CLOSED" {
		t.Fatalf("Invoice status after closing = %q, want CLOSED", closed.Status)
	}
}

func TestInvoiceClosingPersistsPendingBeforeRequestingStockConsumption(t *testing.T) {
	var billingPool *pgxpool.Pool
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/products" {
			_, _ = response.Write([]byte(`[{"id":1,"code":"LAP01","description":"Laptop","stock":5}]`))
			return
		}
		var status string
		var closingPending bool
		if err := billingPool.QueryRow(context.Background(), `SELECT status, closing_pending FROM invoices WHERE number = 1`).Scan(&status, &closingPending); err != nil {
			t.Errorf("read Invoice while inventory is called: %v", err)
		}
		if status != "OPEN" || !closingPending {
			t.Errorf("Invoice while inventory is called = %s / pending %t, want OPEN / true", status, closingPending)
		}
		_, _ = response.Write([]byte(`{"problems":[]}`))
	}))
	t.Cleanup(inventory.Close)
	router, pool := newTestRouterWithPool(t, inventory.URL)
	billingPool = pool
	performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":1}]}`)

	response := performRequest(router, http.MethodPost, "/invoices/0001/close", "")
	if response.Code != http.StatusOK {
		t.Fatalf("POST /invoices/0001/close status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
}

func TestPendingInvoiceClosingCanRetryWithoutPreparingOrPrintingAgain(t *testing.T) {
	consumptionAttempts := 0
	var firstBody map[string]any
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/products" {
			_, _ = response.Write([]byte(`[{"id":1,"code":"LAP01","description":"Laptop","stock":5}]`))
			return
		}
		consumptionAttempts++
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode Stock Consumption attempt: %v", err)
		}
		if consumptionAttempts == 1 {
			firstBody = body
			response.WriteHeader(http.StatusServiceUnavailable)
			_, _ = response.Write([]byte(`{"error":"unavailable"}`))
			return
		}
		if !reflect.DeepEqual(body, firstBody) {
			t.Errorf("retried Stock Consumption body = %#v, want %#v", body, firstBody)
		}
		_, _ = response.Write([]byte(`{"problems":[]}`))
	}))
	t.Cleanup(inventory.Close)
	router := newTestRouter(t, inventory.URL)
	performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":2}]}`)

	firstClose := performRequest(router, http.MethodPost, "/invoices/0001/close", "")
	if firstClose.Code != http.StatusBadGateway {
		t.Fatalf("first POST close status = %d, want 502; body = %s", firstClose.Code, firstClose.Body.String())
	}
	getResponse := performRequest(router, http.MethodGet, "/invoices/0001", "")
	var pending invoice.Invoice
	decodeResponse(t, getResponse, &pending)
	if pending.Status != "OPEN" {
		t.Fatalf("Invoice after failed close = %q, want OPEN", pending.Status)
	}
	prepareResponse := performRequest(router, http.MethodPost, "/invoices/0001/prepare-print", "")
	if prepareResponse.Code != http.StatusConflict {
		t.Fatalf("prepare pending Invoice status = %d, want 409", prepareResponse.Code)
	}

	secondClose := performRequest(router, http.MethodPost, "/invoices/0001/close", "")
	if secondClose.Code != http.StatusOK || consumptionAttempts != 2 {
		t.Fatalf("retried POST close = %d with %d attempts, want 200 with 2 attempts", secondClose.Code, consumptionAttempts)
	}
}

func TestRepeatingACompletedInvoiceClosingDoesNotCallInventoryAgain(t *testing.T) {
	consumptionAttempts := 0
	inventory := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/products" {
			_, _ = response.Write([]byte(`[{"id":1,"code":"LAP01","description":"Laptop","stock":5}]`))
			return
		}
		consumptionAttempts++
		_, _ = response.Write([]byte(`{"problems":[]}`))
	}))
	t.Cleanup(inventory.Close)
	router := newTestRouter(t, inventory.URL)
	performRequest(router, http.MethodPost, "/invoices", `{"lines":[{"productCode":"LAP01","quantity":2}]}`)

	firstClose := performRequest(router, http.MethodPost, "/invoices/0001/close", "")
	secondClose := performRequest(router, http.MethodPost, "/invoices/0001/close", "")
	if firstClose.Code != http.StatusOK || secondClose.Code != http.StatusOK || consumptionAttempts != 1 {
		t.Fatalf("repeated POST close = %d / %d with %d Stock Consumptions, want 200 / 200 with one", firstClose.Code, secondClose.Code, consumptionAttempts)
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
	router, _ := newTestRouterWithPool(t, inventoryURL)
	return router
}

func newTestRouterWithPool(t *testing.T, inventoryURL string) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := newTestPool(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoice.RegisterRoutes(router, pool, inventoryURL, http.DefaultClient)
	return router, pool
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
