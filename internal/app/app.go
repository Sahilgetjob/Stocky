package app

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Sahilgetjob/stocky-backend/internal/handlers"
	"github.com/Sahilgetjob/stocky-backend/internal/models"
	"github.com/Sahilgetjob/stocky-backend/internal/pricing"
	"github.com/Sahilgetjob/stocky-backend/internal/util"
)

func mustEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func Run() error {
	_ = godotenv.Load()

	loc, err := time.LoadLocation(mustEnv("TZ", "Asia/Kolkata"))
	if err != nil {
		return err
	}
	util.SetLocation(loc)

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetLevel(logrus.InfoLevel)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=%s",
		mustEnv("DB_HOST", "localhost"),
		mustEnv("DB_USER", "postgres"),
		mustEnv("DB_PASSWORD", "postgres"),
		mustEnv("DB_NAME", "assignment"),
		mustEnv("DB_PORT", "5432"),
		loc.String(),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.User{}, &models.Reward{}, &models.LedgerEntry{}, &models.StockPrice{}); err != nil {
		return err
	}

	db.FirstOrCreate(&models.User{ID: 1}, &models.User{ID: 1, Name: "Demo User"})

	r := gin.Default()
	h := handlers.New(db)
	h.RegisterRoutes(r)

	go pricing.StartScheduler(db, time.Hour)

	port := mustEnv("APP_PORT", "8080")
	logrus.Infof("listening on :%s", port)
	return r.Run(":" + port)
}
