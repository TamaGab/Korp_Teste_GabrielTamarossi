package product

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tipos e estruturas

var codePattern = regexp.MustCompile(`^[A-Z]{3}[0-9]{2}$`)

type Product struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type productRequest struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Stock       *int   `json:"stock"`
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

type stockProblem struct {
	InventoryProductID int    `json:"inventoryProductId"`
	Reason             string `json:"reason"`
	AvailableStock     *int   `json:"availableStock,omitempty"`
	RequestedQuantity  int    `json:"requestedQuantity"`
}

type handler struct {
	pool *pgxpool.Pool
}

// Registro de rotas

func RegisterRoutes(router gin.IRoutes, pool *pgxpool.Pool) {
	h := handler{pool: pool}
	router.POST("/products", h.create)
	router.GET("/products", h.list)
	router.GET("/products/:id", h.get)
	router.PUT("/products/:id", h.update)
	router.DELETE("/products/:id", h.delete)
	router.POST("/stock/validate", h.validateStock)
	router.POST("/stock/consume", h.consumeStock)
}

// Processamento de estoque

func (h handler) consumeStock(c *gin.Context) {
	var request stockConsumptionRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.InvoiceNumber == "" || !validStockLines(request.Lines) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	transaction, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
		return
	}
	defer func() { _ = transaction.Rollback(c.Request.Context()) }()

	canonicalLines := append([]stockLine(nil), request.Lines...)
	sort.Slice(canonicalLines, func(first, second int) bool {
		return canonicalLines[first].InventoryProductID < canonicalLines[second].InventoryProductID
	})
	payload, err := json.Marshal(canonicalLines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
		return
	}
	payloadHash := sha256.Sum256(payload)
	var insertedInvoiceNumber string
	err = transaction.QueryRow(c.Request.Context(), `
		INSERT INTO stock_consumptions (invoice_number, payload_hash)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		RETURNING invoice_number
	`, request.InvoiceNumber, payloadHash[:]).Scan(&insertedInvoiceNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		var storedHash []byte
		if err := transaction.QueryRow(c.Request.Context(), `
			SELECT payload_hash FROM stock_consumptions WHERE invoice_number = $1
		`, request.InvoiceNumber).Scan(&storedHash); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
			return
		}
		if !bytes.Equal(storedHash, payloadHash[:]) {
			c.JSON(http.StatusConflict, gin.H{"error": "invoice stock consumption payload differs"})
			return
		}
		if err := transaction.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"problems": []stockProblem{}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
		return
	}

	productIDs := make([]int, len(request.Lines))
	for index, line := range request.Lines {
		productIDs[index] = line.InventoryProductID
	}
	rows, err := transaction.Query(c.Request.Context(), `
		SELECT id, stock
		FROM products
		WHERE id = ANY($1)
		ORDER BY id
		FOR UPDATE
	`, productIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
		return
	}

	stockByProductID := make(map[int]int, len(request.Lines))
	for rows.Next() {
		var productID int
		var stock int
		if err := rows.Scan(&productID, &stock); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
			return
		}
		stockByProductID[productID] = stock
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
		return
	}

	problems := stockProblems(request.Lines, stockByProductID)
	if len(problems) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"problems": problems})
		return
	}
	for _, line := range request.Lines {
		if _, err := transaction.Exec(c.Request.Context(), `
			UPDATE products
			SET stock = stock - $1, updated_at = clock_timestamp()
			WHERE id = $2
		`, line.Quantity, line.InventoryProductID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
			return
		}
	}
	if err := transaction.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not consume stock"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"problems": []stockProblem{}})
}

func (h handler) validateStock(c *gin.Context) {
	var request stockRequest
	if err := c.ShouldBindJSON(&request); err != nil || !validStockLines(request.Lines) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	productIDs := make([]int, len(request.Lines))
	for index, line := range request.Lines {
		productIDs[index] = line.InventoryProductID
	}
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, stock
		FROM products
		WHERE id = ANY($1)
	`, productIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not validate stock"})
		return
	}
	defer rows.Close()

	stockByProductID := make(map[int]int, len(request.Lines))
	for rows.Next() {
		var productID int
		var stock int
		if err := rows.Scan(&productID, &stock); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not validate stock"})
			return
		}
		stockByProductID[productID] = stock
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not validate stock"})
		return
	}

	problems := stockProblems(request.Lines, stockByProductID)

	status := http.StatusOK
	if len(problems) > 0 {
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, gin.H{"problems": problems})
}

func stockProblems(lines []stockLine, stockByProductID map[int]int) []stockProblem {
	problems := make([]stockProblem, 0)
	for _, line := range lines {
		availableStock, exists := stockByProductID[line.InventoryProductID]
		if !exists {
			problems = append(problems, stockProblem{
				InventoryProductID: line.InventoryProductID,
				Reason:             "product_not_found",
				RequestedQuantity:  line.Quantity,
			})
			continue
		}
		if availableStock < line.Quantity {
			problems = append(problems, stockProblem{
				InventoryProductID: line.InventoryProductID,
				Reason:             "insufficient_stock",
				AvailableStock:     &availableStock,
				RequestedQuantity:  line.Quantity,
			})
		}
	}
	return problems
}

func validStockLines(lines []stockLine) bool {
	if len(lines) == 0 {
		return false
	}
	seen := make(map[int]struct{}, len(lines))
	for _, line := range lines {
		if line.InventoryProductID <= 0 || line.Quantity <= 0 {
			return false
		}
		if _, exists := seen[line.InventoryProductID]; exists {
			return false
		}
		seen[line.InventoryProductID] = struct{}{}
	}
	return true
}

// Consulta e manutenção de produtos

func (h handler) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	result, err := h.pool.Exec(c.Request.Context(), "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete product"})
		return
	}
	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h handler) get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	var found Product
	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT id, code, description, stock, created_at, updated_at
		FROM products
		WHERE id = $1
	`, id).Scan(
		&found.ID,
		&found.Code,
		&found.Description,
		&found.Stock,
		&found.CreatedAt,
		&found.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get product"})
		return
	}

	found.CreatedAt = found.CreatedAt.UTC()
	found.UpdatedAt = found.UpdatedAt.UTC()
	c.JSON(http.StatusOK, found)
}

func (h handler) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	var request productRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	request.Description = strings.TrimSpace(request.Description)
	if message := validateProduct(request); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	var updated Product
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE products
		SET code = $1, description = $2, stock = $3, updated_at = clock_timestamp()
		WHERE id = $4
		RETURNING id, code, description, stock, created_at, updated_at
	`, request.Code, request.Description, *request.Stock, id).Scan(
		&updated.ID,
		&updated.Code,
		&updated.Description,
		&updated.Stock,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "product code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update product"})
		return
	}

	updated.CreatedAt = updated.CreatedAt.UTC()
	updated.UpdatedAt = updated.UpdatedAt.UTC()
	c.JSON(http.StatusOK, updated)
}

func (h handler) create(c *gin.Context) {
	var request productRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	request.Description = strings.TrimSpace(request.Description)
	if message := validateProduct(request); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	var created Product
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO products (code, description, stock)
		VALUES ($1, $2, $3)
		RETURNING id, code, description, stock, created_at, updated_at
	`, request.Code, request.Description, *request.Stock).Scan(
		&created.ID,
		&created.Code,
		&created.Description,
		&created.Stock,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "product code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create product"})
		return
	}

	created.CreatedAt = created.CreatedAt.UTC()
	created.UpdatedAt = created.UpdatedAt.UTC()
	c.JSON(http.StatusCreated, created)
}

func (h handler) list(c *gin.Context) {
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, code, description, stock, created_at, updated_at
		FROM products
		ORDER BY id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list products"})
		return
	}
	defer rows.Close()

	products, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Product, error) {
		var product Product
		err := row.Scan(
			&product.ID,
			&product.Code,
			&product.Description,
			&product.Stock,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		product.CreatedAt = product.CreatedAt.UTC()
		product.UpdatedAt = product.UpdatedAt.UTC()
		return product, err
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

// Validação de produtos

func validateProduct(request productRequest) string {
	if !codePattern.MatchString(request.Code) {
		return "code must match AAA00"
	}
	if request.Description == "" || len([]rune(request.Description)) > 200 {
		return "description must contain between 1 and 200 characters"
	}
	if request.Stock == nil || *request.Stock < 0 {
		return "stock must be a non-negative integer"
	}
	return ""
}
