// internal/http/router/router.go
package router

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/http/handlers"
	middleware "github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/http/midleware"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/repository"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/service"
)

func Setup(db *gorm.DB, jwtSecret string) *gin.Engine {
	r := gin.Default()

	// aceita uploads grandes (ex.: até 512 MiB)
	r.MaxMultipartMemory = 512 << 20 // 512 MiB

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // ajuste se necessário
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// repos
	relRepo := repository.NewReleaseRepository(db)
	userRepo := repository.NewUserRepository(db)

	// services
	relSvc := service.NewReleaseService(relRepo)
	authSvc := service.NewAuthService(userRepo, jwtSecret)
	userSvc := service.NewUserService(userRepo)

	// handlers
	auth := handlers.AuthHandler{Svc: authSvc}
	rel := handlers.ReleaseHandler{
    Svc:            relSvc,
    FileLocalRoot:  envOr("FILE_LOCAL_ROOT", "/files/firmware"),
    FilePublicBase: envOr("FILE_PUBLIC_BASE", "https://file-serve.api-castilho.com.br/firmware/"),
}

	user := handlers.UserHandler{Svc: userSvc}

	// auth pública
	r.POST("/api/auth/login", auth.Login)
	r.POST("/api/users", user.Create)

	// público
	r.GET("/api/releases", rel.List)
	r.GET("/api/releases/:id", rel.Get)

	// protegido
	protected := r.Group("/api")
	protected.Use(middleware.JWT(jwtSecret))
	{
		ed := protected.Group("/releases")
		ed.Use(middleware.RequireRole("admin", "editor"))
		// Create já aceita JSON ou multipart com upload (campo "data" + "file")
		ed.POST("", rel.Create)
		ed.PUT("/:id", rel.Update)
		ed.DELETE("/:id", middleware.RequireRole("admin"), rel.Delete)
	}

	return r
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
