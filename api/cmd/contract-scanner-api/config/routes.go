package config

import (
	"net/http"

	"contract-scanner/internal/handler"
	"contract-scanner/internal/infra/auth"
	"contract-scanner/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Routes(
	uploadHandler *handler.UploadHandler,
	analyseHandler *handler.AnalyseHandler,
	authHandler *handler.AuthHandler,
	jwtSvc auth.JWTService,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		// Public auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.GET("/me", middleware.JWTAuth(jwtSvc), authHandler.Me)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.JWTAuth(jwtSvc))
		{
			uploads := protected.Group("/uploads")
			{
				uploads.POST("", uploadHandler.Upload)
			}

			analyses := protected.Group("/analyses")
			{
				analyses.GET("", analyseHandler.List)
				analyses.POST("/:id/process", analyseHandler.Process)
				analyses.DELETE("/:id", analyseHandler.Delete)
			}
		}
	}

	return r
}
