package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAcceptsLocalFrontendHostAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware("http://localhost:4200"))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, origin := range []string{"http://localhost:4200", "http://127.0.0.1:4200"} {
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request.Header.Set("Origin", origin)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if allowed := response.Header().Get("Access-Control-Allow-Origin"); allowed != origin {
			t.Errorf("Origin %q allowed as %q, want exact requesting Origin", origin, allowed)
		}
	}
}

func TestCORSRejectsUnconfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware("http://localhost:4200"))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("Origin", "http://example.com")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if allowed := response.Header().Get("Access-Control-Allow-Origin"); allowed != "" {
		t.Fatalf("unconfigured Origin allowed as %q", allowed)
	}
}
