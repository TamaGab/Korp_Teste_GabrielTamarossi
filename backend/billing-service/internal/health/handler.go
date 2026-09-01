package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Handler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		pingContext, cancelPing := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancelPing()
		if err := pool.Ping(pingContext); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "degraded", "service": "ok", "database": "unavailable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ok", "database": "ok"})
	}
}
