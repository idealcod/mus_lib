package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "music-library/docs"
	"music-library/internal/api"
	"music-library/internal/repository"
	"music-library/internal/service"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting application...")
	logger.Debug("Initializing logger")

	logger.Debug("Loading .env file")
	err = godotenv.Load()
	if err != nil {
		logger.Warn("Failed to load .env file, using default values", zap.Error(err))
	}

	dbHost := getEnv("DB_HOST", "db")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "123456")
	dbName := getEnv("DB_NAME", "music_library")

	logger.Debug("Fetching environment variables", zap.String("DB_HOST", dbHost), zap.String("DB_PORT", dbPort))

	sqlxConnStr := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + " password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"
	logger.Info("Connection string for sqlx", zap.String("sqlxConnStr", sqlxConnStr))

	migrateConnStr := "postgres://" + dbUser + ":" + dbPassword + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"
	logger.Info("Connection string for migrate", zap.String("migrateConnStr", migrateConnStr))

	logger.Debug("Attempting to connect to database")
	var db *sqlx.DB
	for i := 0; i < 10; i++ {
		db, err = sqlx.Connect("postgres", sqlxConnStr)
		if err == nil {
			if err := db.Ping(); err == nil {
				break
			}
		}
		logger.Warn("Failed to connect to database, retrying...", zap.Error(err), zap.Int("attempt", i+1))
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		logger.Fatal("Failed to connect to database after retries", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Successfully connected to database")

	migrationURL := "file:///app/migrations"
	logger.Info("Attempting to initialize migrations with URL", zap.String("migrationURL", migrationURL))
	logger.Debug("Running migrations")

	migrations, err := migrate.New(migrationURL, migrateConnStr)
	if err != nil {
		logger.Fatal("Failed to initialize migrations", zap.Error(err))
	}

	if err := migrations.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Info("No migrations to apply")
		} else {
			logger.Error("Migration failed", zap.Error(err))
			logger.Fatal("Application cannot start due to migration failure", zap.Error(err))
		}
	} else {
		logger.Info("Migrations applied successfully")
		migrations.Close()
	}

	logger.Debug("Initializing dependencies")
	repo := repository.NewPostgresRepository(db)
	svc := service.NewMusicService(repo, logger, &http.Client{})
	handler := api.NewHandler(svc, logger)

	logger.Debug("Configuring Gin router")
	r := gin.Default()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/songs", handler.GetSongs)
	r.POST("/songs", handler.AddSong)
	r.GET("/songs/:id/verses", handler.GetVerses)
	r.PUT("/songs/:id", handler.UpdateSong)
	r.DELETE("/songs/:id", handler.DeleteSong)
	r.POST("/songs/truncate", handler.TruncateSongs)

	port := getEnv("PORT", "8080")
	logger.Info("Starting server", zap.String("port", port))
	logger.Debug("Server starting on port", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
