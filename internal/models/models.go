package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Name      string `json:"name"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Reward struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"userId" gorm:"index"`
	Symbol         string    `json:"symbol" gorm:"index"`
	Units          string    `json:"units" gorm:"type:numeric(18,6)"`
	EventTime      time.Time `json:"eventTime" gorm:"index"`
	IdempotencyKey string    `json:"idempotencyKey" gorm:"uniqueIndex"`
	CreatedAt      time.Time
}

type LedgerEntry struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"userId" gorm:"index"`
	Account   string    `json:"account" gorm:"index"`
	Symbol    *string   `json:"symbol" gorm:"index"`
	Units     *string   `json:"units" gorm:"type:numeric(18,6)"`
	INRAmount *string   `json:"inrAmount" gorm:"type:numeric(18,4)"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`
	Meta      string    `json:"meta" gorm:"type:jsonb;default:'{}'"`
}

type StockPrice struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Symbol    string    `json:"symbol" gorm:"index"`
	Price     string    `json:"price" gorm:"type:numeric(18,4)"`
	AsOf      time.Time `json:"asOf" gorm:"index"`
	CreatedAt time.Time
}
