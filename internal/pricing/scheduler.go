package pricing

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Sahilgetjob/stocky-backend/internal/models"
	"github.com/Sahilgetjob/stocky-backend/internal/util"
)

func StartScheduler(db *gorm.DB, interval time.Duration) {
	go seedOnce(db)
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		if err := updatePrices(db); err != nil {
			logrus.WithError(err).Warn("price update failed")
		}
	}
}

func seedOnce(db *gorm.DB) {
	time.Sleep(2 * time.Second)
	_ = updatePrices(db)
}

func updatePrices(db *gorm.DB) error {
	var syms []string
	if err := db.Model(&models.Reward{}).Distinct().Pluck("symbol", &syms).Error; err != nil {
		return err
	}
	if len(syms) == 0 {
		syms = []string{"RELIANCE", "TCS", "INFY"}
	}
	for _, s := range syms {
		price := nextPrice(db, s)
		sp := models.StockPrice{
			Symbol: s,
			Price:  fmt.Sprintf("%.4f", price),
			AsOf:   util.Now(),
		}
		if err := db.Create(&sp).Error; err != nil {
			return err
		}
	}
	logrus.Infof("updated prices for %d symbols", len(syms))
	return nil
}

func nextPrice(db *gorm.DB, symbol string) float64 {
	var last models.StockPrice
	base := 1000.0
	if err := db.Where("symbol = ?", symbol).Order("as_of DESC").First(&last).Error; err == nil {
		var p float64
		fmt.Sscanf(last.Price, "%f", &p)
		base = p
	} else {
		base = 800 + rand.Float64()*600
	}
	delta := (rand.Float64()*0.06 - 0.03) * base
	next := base + delta
	if next < 10 {
		next = 10
	}
	return next
}
