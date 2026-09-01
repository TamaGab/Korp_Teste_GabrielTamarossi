package product

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

type stockValidationRequest struct {
	Lines []stockValidationLine `json:"lines"`
}

type stockValidationLine struct {
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

func RegisterRoutes(router gin.IRoutes, pool *pgxpool.Pool) {
	h := handler{pool: pool}
	router.POST("/products", h.create)
	router.GET("/products", h.list)
	router.GET("/products/:id", h.get)
	router.PUT("/products/:id", h.update)
	router.DELETE("/products/:id", h.delete)
	router.POST("/stock/validate", h.validateStock)
}

func (h handler) validateStock(c *gin.Context) {
	var request stockValidationRequest
	if err := c.ShouldBindJSON(&request); err != nil || !validStockValidation(request) {
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

	problems := make([]stockProblem, 0)
	for _, line := range request.Lines {
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

	status := http.StatusOK
	if len(problems) > 0 {
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, gin.H{"problems": problems})
}

func validStockValidation(request stockValidationRequest) bool {
	if len(request.Lines) == 0 {
		return false
	}
	seen := make(map[int]struct{}, len(request.Lines))
	for _, line := range request.Lines {
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
