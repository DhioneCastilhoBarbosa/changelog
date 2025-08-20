package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/config"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/db"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/http/handlers"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/http/router"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
)

func main() {
	// Tenta carregar o .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Nenhum arquivo .env encontrado, usando variáveis de ambiente do sistema.")
	}

	// Valida variáveis obrigatórias
	requiredVars := []string{"DATABASE_URL", "JWT_SECRET", "PORT"}
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("❌ Variável de ambiente obrigatória ausente: %s", v)
		}
	}

	cfg := config.Load()
	gormDB := db.MustOpen(cfg.DatabaseURL)

	if err := gormDB.AutoMigrate(
		&models.User{},
		&models.Release{},
		&models.ReleaseModule{},
		&models.ChangelogEntry{},
	); err != nil {
		log.Fatal(err)
	}

	// opcional
	handlers.SeedAdmin(gormDB)

	r := router.Setup(gormDB, cfg.JWTSecret)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
