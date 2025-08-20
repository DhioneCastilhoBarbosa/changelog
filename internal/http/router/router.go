// internal/http/router/router.go
package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/http/handlers"
	middleware "github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/http/midleware"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/repository"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/service"
)

func Setup(db *gorm.DB, jwtSecret string) *gin.Engine {
	r := gin.Default()

	// repos
	relRepo := repository.NewReleaseRepository(db)
	userRepo := repository.NewUserRepository(db)

	// services
	relSvc := service.NewReleaseService(relRepo)
	authSvc := service.NewAuthService(userRepo, jwtSecret)
	userSvc := service.NewUserService(userRepo)

	// handlers
	auth := handlers.AuthHandler{Svc: authSvc}
	rel := handlers.ReleaseHandler{Svc: relSvc}
	user := handlers.UserHandler{Svc: userSvc}

	// auth
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
		ed.POST("", rel.Create)
		ed.PUT("/:id", rel.Update)
		ed.DELETE("/:id", middleware.RequireRole("admin"), rel.Delete)
	}
	return r
}
