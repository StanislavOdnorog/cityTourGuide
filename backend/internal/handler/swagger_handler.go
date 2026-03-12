package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/saas/city-stories-guide/backend/api"
)

// RegisterSwagger mounts the OpenAPI spec and Swagger UI routes on the router.
// - GET /api/openapi.yaml  — serves the raw OpenAPI 3.0 spec
// - GET /api/docs/*any     — serves Swagger UI pointed at the spec
func RegisterSwagger(r *gin.Engine) {
	r.GET("/api/openapi.yaml", serveSpec)
	r.GET("/api/docs/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/api/openapi.yaml"),
	))
}

func serveSpec(c *gin.Context) {
	data, err := api.SpecFS.ReadFile("openapi.yaml")
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "failed to load spec")
		return
	}
	c.Data(http.StatusOK, "application/yaml", data)
}
