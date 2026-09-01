package invoice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var productCodePattern = regexp.MustCompile(`^[A-Z]{3}[0-9]{2}$`)

type Invoice struct {
	Number         string        `json:"number"`
	Status         string        `json:"status"`
	ClosingPending bool          `json:"closingPending"`
	Lines          []InvoiceLine `json:"lines,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
}

type InvoiceLine struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
}

type invoiceRequest struct {
	Lines []invoiceLineRequest `json:"lines"`
}

type invoiceLineRequest struct {
	ProductCode string `json:"productCode"`
	Quantity    int    `json:"quantity"`
}

type inventoryProduct struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type validatedLine struct {
	InvoiceLine
	InventoryProductID int
}

type stockRequest struct {
	Lines []stockLine `json:"lines"`
}

type stockConsumptionRequest struct {
	InvoiceNumber string      `json:"invoiceNumber"`
	Lines         []stockLine `json:"lines"`
}

type stockLine struct {
	InventoryProductID int `json:"inventoryProductId"`
	Quantity           int `json:"quantity"`
}

type stockResponse struct {
	Problems []inventoryStockProblem `json:"problems"`
}

type inventoryStockProblem struct {
	InventoryProductID int    `json:"inventoryProductId"`
	Reason             string `json:"reason"`
	AvailableStock     *int   `json:"availableStock,omitempty"`
	RequestedQuantity  int    `json:"requestedQuantity"`
}

type printProblem struct {
	ProductCode       string `json:"productCode"`
	Reason            string `json:"reason"`
	AvailableStock    *int   `json:"availableStock,omitempty"`
	RequestedQuantity int    `json:"requestedQuantity"`
}

type printableInvoice struct {
	Number string
	Lines  []InvoiceLine
}

var printableInvoiceTemplate = template.Must(template.New("invoice").Parse(`<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<title>Nota Fiscal {{.Number}}</title>
<style>
body{font-family:Arial,sans-serif;margin:32px;color:#111}h1{margin:0 0 8px}table{border-collapse:collapse;margin-top:24px;width:100%}th,td{border:1px solid #bbb;padding:8px;text-align:left}th:last-child,td:last-child{text-align:right}
</style>
</head>
<body>
<h1>Nota Fiscal</h1>
<p>Número: {{.Number}}</p>
<table>
<thead><tr><th>Código do produto</th><th>Descrição</th><th>Quantidade</th></tr></thead>
<tbody>{{range .Lines}}<tr><td>{{.Code}}</td><td>{{.Description}}</td><td>{{.Quantity}}</td></tr>{{end}}</tbody>
</table>
</body>
</html>`))

type handler struct {
	pool         *pgxpool.Pool
	inventoryURL string
	httpClient   *http.Client
}

func RegisterRoutes(router gin.IRoutes, pool *pgxpool.Pool, inventoryURL string, httpClient *http.Client) {
	h := handler{
		pool:         pool,
		inventoryURL: strings.TrimRight(inventoryURL, "/"),
		httpClient:   httpClient,
	}
	router.POST("/invoices", h.create)
	router.GET("/invoices", h.list)
	router.GET("/invoices/:number", h.get)
	router.POST("/invoices/:number/prepare-print", h.preparePrint)
	router.POST("/invoices/:number/close", h.close)
}

func (h handler) close(c *gin.Context) {
	number, err := strconv.ParseInt(c.Param("number"), 10, 64)
	if err != nil || number <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	transaction, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
		return
	}
	defer func() { _ = transaction.Rollback(c.Request.Context()) }()

	var status string
	var closingPending bool
	if err := transaction.QueryRow(c.Request.Context(), `
		SELECT status, closing_pending
		FROM invoices
		WHERE number = $1
		FOR UPDATE
	`, number).Scan(&status, &closingPending); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
		return
	}
	if status == "CLOSED" {
		if err := transaction.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"number": fmt.Sprintf("%04d", number), "status": "CLOSED"})
		return
	}

	rows, err := transaction.Query(c.Request.Context(), `
		SELECT inventory_product_id, quantity
		FROM invoice_lines
		WHERE invoice_number = $1
		ORDER BY inventory_product_id
	`, number)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
		return
	}
	lines := make([]stockLine, 0)
	for rows.Next() {
		var line stockLine
		if err := rows.Scan(&line.InventoryProductID, &line.Quantity); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
			return
		}
		lines = append(lines, line)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
		return
	}
	if !closingPending {
		if _, err := transaction.Exec(c.Request.Context(), `
			UPDATE invoices SET closing_pending = TRUE WHERE number = $1
		`, number); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
			return
		}
	}
	if err := transaction.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
		return
	}

	consumption, inventoryStatus, err := h.consumeStock(c.Request.Context(), fmt.Sprintf("%04d", number), lines)
	if err != nil || (inventoryStatus != http.StatusOK && inventoryStatus != http.StatusUnprocessableEntity && inventoryStatus != http.StatusConflict) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "inventory unavailable"})
		return
	}
	if inventoryStatus != http.StatusOK {
		c.JSON(inventoryStatus, consumption)
		return
	}

	if _, err := h.pool.Exec(c.Request.Context(), `
		UPDATE invoices
		SET status = 'CLOSED', closing_pending = FALSE
		WHERE number = $1
	`, number); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close invoice"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"number": fmt.Sprintf("%04d", number), "status": "CLOSED"})
}

func (h handler) consumeStock(ctx context.Context, invoiceNumber string, lines []stockLine) (stockResponse, int, error) {
	payload, err := json.Marshal(stockConsumptionRequest{InvoiceNumber: invoiceNumber, Lines: lines})
	if err != nil {
		return stockResponse{}, 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, h.inventoryURL+"/stock/consume", bytes.NewReader(payload))
	if err != nil {
		return stockResponse{}, 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := h.httpClient.Do(request)
	if err != nil {
		return stockResponse{}, 0, err
	}
	defer response.Body.Close()

	var consumption stockResponse
	if err := json.NewDecoder(response.Body).Decode(&consumption); err != nil {
		return stockResponse{}, response.StatusCode, fmt.Errorf("decode stock consumption: %w", err)
	}
	return consumption, response.StatusCode, nil
}

func (h handler) preparePrint(c *gin.Context) {
	number, err := strconv.ParseInt(c.Param("number"), 10, 64)
	if err != nil || number <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	transaction, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare invoice print"})
		return
	}
	defer func() { _ = transaction.Rollback(c.Request.Context()) }()

	var status string
	var closingPending bool
	if err := transaction.QueryRow(c.Request.Context(), `
		SELECT status, closing_pending
		FROM invoices
		WHERE number = $1
		FOR SHARE
	`, number).Scan(&status, &closingPending); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare invoice print"})
		return
	}
	if status != "OPEN" || closingPending {
		c.JSON(http.StatusConflict, gin.H{"error": "invoice cannot be prepared for print"})
		return
	}

	rows, err := transaction.Query(c.Request.Context(), `
		SELECT inventory_product_id, product_code, product_description, quantity
		FROM invoice_lines
		WHERE invoice_number = $1
		ORDER BY inventory_product_id
	`, number)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare invoice print"})
		return
	}
	defer rows.Close()

	validationLines := make([]stockLine, 0)
	printLines := make([]InvoiceLine, 0)
	productCodes := make(map[int]string)
	for rows.Next() {
		var productID int
		var line InvoiceLine
		if err := rows.Scan(&productID, &line.Code, &line.Description, &line.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare invoice print"})
			return
		}
		validationLines = append(validationLines, stockLine{InventoryProductID: productID, Quantity: line.Quantity})
		printLines = append(printLines, line)
		productCodes[productID] = line.Code
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare invoice print"})
		return
	}

	validation, inventoryStatus, err := h.validateStock(c.Request.Context(), validationLines)
	if err != nil || (inventoryStatus != http.StatusOK && inventoryStatus != http.StatusUnprocessableEntity) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "inventory unavailable"})
		return
	}
	if inventoryStatus == http.StatusUnprocessableEntity {
		problems := make([]printProblem, 0, len(validation.Problems))
		for _, problem := range validation.Problems {
			problems = append(problems, printProblem{
				ProductCode:       productCodes[problem.InventoryProductID],
				Reason:            problem.Reason,
				AvailableStock:    problem.AvailableStock,
				RequestedQuantity: problem.RequestedQuantity,
			})
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":    "print preparation failed",
			"problems": problems,
		})
		return
	}

	var html bytes.Buffer
	if err := printableInvoiceTemplate.Execute(&html, printableInvoice{
		Number: fmt.Sprintf("%04d", number),
		Lines:  printLines,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare invoice print"})
		return
	}
	if err := transaction.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not prepare invoice print"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"html": html.String()})
}

func (h handler) validateStock(ctx context.Context, lines []stockLine) (stockResponse, int, error) {
	payload, err := json.Marshal(stockRequest{Lines: lines})
	if err != nil {
		return stockResponse{}, 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, h.inventoryURL+"/stock/validate", bytes.NewReader(payload))
	if err != nil {
		return stockResponse{}, 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := h.httpClient.Do(request)
	if err != nil {
		return stockResponse{}, 0, err
	}
	defer response.Body.Close()

	var validation stockResponse
	if err := json.NewDecoder(response.Body).Decode(&validation); err != nil {
		return stockResponse{}, response.StatusCode, fmt.Errorf("decode stock validation: %w", err)
	}
	return validation, response.StatusCode, nil
}

func (h handler) list(c *gin.Context) {
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT number, status, closing_pending, created_at
		FROM invoices
		ORDER BY number DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list invoices"})
		return
	}
	defer rows.Close()

	invoices := make([]Invoice, 0)
	for rows.Next() {
		var number int64
		var current Invoice
		if err := rows.Scan(&number, &current.Status, &current.ClosingPending, &current.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list invoices"})
			return
		}
		current.Number = fmt.Sprintf("%04d", number)
		current.CreatedAt = current.CreatedAt.UTC()
		invoices = append(invoices, current)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list invoices"})
		return
	}

	c.JSON(http.StatusOK, invoices)
}

func (h handler) get(c *gin.Context) {
	number, err := strconv.ParseInt(c.Param("number"), 10, 64)
	if err != nil || number <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	current := Invoice{Number: fmt.Sprintf("%04d", number)}
	if err := h.pool.QueryRow(c.Request.Context(), `
		SELECT status, closing_pending, created_at
		FROM invoices
		WHERE number = $1
	`, number).Scan(&current.Status, &current.ClosingPending, &current.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get invoice"})
		return
	}
	current.CreatedAt = current.CreatedAt.UTC()

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT product_code, product_description, quantity
		FROM invoice_lines
		WHERE invoice_number = $1
		ORDER BY inventory_product_id
	`, number)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get invoice"})
		return
	}
	defer rows.Close()

	current.Lines = make([]InvoiceLine, 0)
	for rows.Next() {
		var line InvoiceLine
		if err := rows.Scan(&line.Code, &line.Description, &line.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get invoice"})
			return
		}
		current.Lines = append(current.Lines, line)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get invoice"})
		return
	}

	c.JSON(http.StatusOK, current)
}

func (h handler) create(c *gin.Context) {
	var request invoiceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if message := validateInvoice(request); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	products, err := h.listProducts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not validate products"})
		return
	}
	productsByCode := make(map[string]inventoryProduct, len(products))
	for _, product := range products {
		productsByCode[product.Code] = product
	}

	lines := make([]validatedLine, 0, len(request.Lines))
	for _, requestedLine := range request.Lines {
		product, exists := productsByCode[requestedLine.ProductCode]
		if !exists {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":       "product not found",
				"productCode": requestedLine.ProductCode,
			})
			return
		}
		lines = append(lines, validatedLine{
			InvoiceLine: InvoiceLine{
				Code:        product.Code,
				Description: product.Description,
				Quantity:    requestedLine.Quantity,
			},
			InventoryProductID: product.ID,
		})
	}

	transaction, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create invoice"})
		return
	}
	defer func() { _ = transaction.Rollback(c.Request.Context()) }()

	var number int64
	var createdAt time.Time
	if err := transaction.QueryRow(c.Request.Context(), `
		INSERT INTO invoices DEFAULT VALUES
		RETURNING number, created_at
	`).Scan(&number, &createdAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create invoice"})
		return
	}

	for _, line := range lines {
		if _, err := transaction.Exec(c.Request.Context(), `
			INSERT INTO invoice_lines (
				invoice_number, inventory_product_id, product_code, product_description, quantity
			) VALUES ($1, $2, $3, $4, $5)
		`, number, line.InventoryProductID, line.Code, line.Description, line.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create invoice"})
			return
		}
	}

	if err := transaction.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create invoice"})
		return
	}

	responseLines := make([]InvoiceLine, len(lines))
	for index, line := range lines {
		responseLines[index] = line.InvoiceLine
	}
	c.JSON(http.StatusCreated, Invoice{
		Number:    fmt.Sprintf("%04d", number),
		Status:    "OPEN",
		Lines:     responseLines,
		CreatedAt: createdAt.UTC(),
	})
}

func (h handler) listProducts(ctx context.Context) ([]inventoryProduct, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, h.inventoryURL+"/products", nil)
	if err != nil {
		return nil, err
	}
	response, err := h.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inventory returned status %d", response.StatusCode)
	}

	var products []inventoryProduct
	if err := json.NewDecoder(response.Body).Decode(&products); err != nil {
		return nil, fmt.Errorf("decode inventory products: %w", err)
	}
	return products, nil
}

func validateInvoice(request invoiceRequest) string {
	if len(request.Lines) == 0 {
		return "invoice must contain at least one line"
	}

	products := make(map[string]struct{}, len(request.Lines))
	for _, line := range request.Lines {
		if !productCodePattern.MatchString(line.ProductCode) {
			return "productCode must match AAA00"
		}
		if line.Quantity <= 0 {
			return "quantity must be a positive integer"
		}
		if _, exists := products[line.ProductCode]; exists {
			return "invoice cannot contain duplicate products"
		}
		products[line.ProductCode] = struct{}{}
	}
	return ""
}
