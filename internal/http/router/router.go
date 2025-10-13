package router

import (
	"os"
	"strconv"
	"strings"
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
        AllowOrigins:     []string{"http://localhost:5173",
    "https://changelog.intelbras-cve-pro.com.br"},
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
        FilePublicBase: strings.TrimRight(envOr("FILE_PUBLIC_BASE", "https://files.seudominio.com/firmware"), "/"),
        FileServerBase: strings.TrimRight(envOr("FILE_SERVER_BASE", "https://files.seudominio.com/firmware"), "/"),
        FileServerUser: envOr("FILE_SERVER_USER", "uploader"),
        FileServerPass: envOr("FILE_SERVER_PASS", ""),
        HTTPTimeout:    envDur("HTTP_TIMEOUT", 120*time.Second), // aceita "120s" ou "120"
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

        // Create aceita JSON ou multipart (campo "data" + "file"), e usa DAV PUT
        ed.POST("", rel.Create)
        ed.PUT("/:id", rel.Update)

        // Apaga release; antes tenta DAV DELETE para cada link do release
        ed.DELETE("/:id", middleware.RequireRole("admin"), rel.Delete)

        // Apagar um arquivo avulso do file-server (URL ou path em JSON)
        ed.DELETE("/file", middleware.RequireRole("admin"), rel.DeleteFile)
    }

    return r
}

func envOr(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}

// HTTP_TIMEOUT: "120s" OU "120" (segundos)
func envDur(k string, def time.Duration) time.Duration {
    v := os.Getenv(k)
    if v == "" { return def }
    if d, err := time.ParseDuration(v); err == nil { return d }
    if n, err := strconv.Atoi(v); err == nil { return time.Duration(n) * time.Second }
    return def
}
