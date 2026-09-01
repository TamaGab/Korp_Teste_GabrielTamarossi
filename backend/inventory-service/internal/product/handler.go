package product

import (
	"errors"
	"net/http"
	"regexp"
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

type createProductRequest struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Stock       *int   `json:"stock"`
}

type handler struct {
	pool *pgxpool.Pool
}

func RegisterRoutes(router gin.IRoutes, pool *pgxpool.Pool) {
	h := handler{pool: pool}
	router.POST("/products", h.create)
	router.GET("/products", h.list)
}

func (h handler) create(c *gin.Context) {
	var request createProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	request.Description = strings.TrimSpace(request.Description)
	if message := validateCreate(request); message != "" {
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

func validateCreate(request createProductRequest) string {
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
