package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Sahilgetjob/stocky-backend/internal/models"
	"github.com/Sahilgetjob/stocky-backend/internal/util"
)

// customTime allows parsing timestamps with or without timezone
type customTime struct {
	time.Time
}

func (ct *customTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) > 0 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	// Try RFC3339 with timezone first
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		ct.Time = t
		return nil
	}

	// Try ISO8601 without timezone (treat as UTC)
	t, err = time.Parse("2006-01-02T15:04:05.000000", s)
	if err == nil {
		ct.Time = t
		return nil
	}

	return fmt.Errorf("cannot parse timestamp %q", s)
}

type Handler struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/reward", h.postReward)
	r.GET("/today-stocks/:userId", h.getTodayStocks)
	r.GET("/historical-inr/:userId", h.getHistoricalINR)
	r.GET("/stats/:userId", h.getStats)
	r.GET("/portfolio/:userId", h.getPortfolio)
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
}

type postRewardReq struct {
	IdempotencyKey string     `json:"idempotencyKey"`
	UserID         uint       `json:"userId" binding:"required"`
	Symbol         string     `json:"symbol" binding:"required"`
	Units          string     `json:"units" binding:"required"`
	Timestamp      customTime `json:"timestamp"`
}

const (
	brokeragePct = 0.005
	sttPct       = 0.001
	gstPct       = 0.18
)

func (h *Handler) postReward(c *gin.Context) {
	var req postRewardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate units: must be positive
	var fUnits float64
	_, _ = fmt.Sscanf(req.Units, "%f", &fUnits)
	if fUnits <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "units must be positive"})
		return
	}

	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	if req.Timestamp.IsZero() {
		req.Timestamp.Time = util.Now()
	}

	// idempotency
	if req.IdempotencyKey != "" {
		var existing models.Reward
		if err := h.db.Where("idempotency_key = ?", req.IdempotencyKey).First(&existing).Error; err == nil {
			c.JSON(http.StatusOK, gin.H{"id": existing.ID, "status": "duplicate_ignored"})
			return
		}
	}

	var price models.StockPrice
	if err := h.db.Where("symbol = ?", req.Symbol).Order("as_of DESC").First(&price).Error; err != nil {
		price.Price = "1000.0000"
	}

	toDec := func(s string) float64 {
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}
	_, _ = fmt.Sscanf(req.Units, "%f", &fUnits)
	notional := toDec(price.Price) * fUnits
	brokerage := notional * brokeragePct
	stt := notional * sttPct
	gst := brokerage * gstPct
	totalCost := notional + brokerage + stt + gst

	// transaction
	err := h.db.Transaction(func(tx *gorm.DB) error {
		rw := models.Reward{
			UserID:         req.UserID,
			Symbol:         req.Symbol,
			Units:          req.Units,
			EventTime:      req.Timestamp.Time,
			IdempotencyKey: req.IdempotencyKey,
		}
		if err := tx.Create(&rw).Error; err != nil {
			return err
		}

		sym := req.Symbol
		unitsStr := fmt.Sprintf("%.6f", fUnits)
		notionalStr := fmt.Sprintf("%.4f", notional)
		brokerageStr := fmt.Sprintf("%.4f", brokerage)
		sttStr := fmt.Sprintf("%.4f", stt)
		gstStr := fmt.Sprintf("%.4f", gst)
		totalStr := fmt.Sprintf("%.4f", totalCost)

		entries := []models.LedgerEntry{
			{UserID: req.UserID, Account: "stock_units", Symbol: &sym, Units: &unitsStr, INRAmount: &notionalStr, CreatedAt: util.Now()},
			{UserID: req.UserID, Account: "cash", INRAmount: &totalStr, CreatedAt: util.Now(), Meta: `{"reason":"purchase"}`},
			{UserID: req.UserID, Account: "brokerage", INRAmount: &brokerageStr, CreatedAt: util.Now()},
			{UserID: req.UserID, Account: "stt", INRAmount: &sttStr, CreatedAt: util.Now()},
			{UserID: req.UserID, Account: "gst", INRAmount: &gstStr, CreatedAt: util.Now()},
		}
		if err := tx.Create(&entries).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate idempotency key"})
			return
		}
		logrus.WithError(err).Error("reward txn failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"fees": gin.H{
			"brokeragePct": brokeragePct,
			"sttPct":       sttPct,
			"gstPct":       gstPct,
		},
	})
}

func (h *Handler) getTodayStocks(c *gin.Context) {
	userId := c.Param("userId")
	start, end := util.TodayRange()
	var rewards []models.Reward
	if err := h.db.Where("user_id = ? AND event_time >= ? AND event_time < ?", userId, start, end).Order("event_time ASC").Find(&rewards).Error; err != nil {
		c.JSON(500, gin.H{"error": "db"})
		return
	}
	c.JSON(200, gin.H{"count": len(rewards), "items": rewards})
}

func (h *Handler) getHistoricalINR(c *gin.Context) {
	userId := c.Param("userId")
	start, _ := util.TodayRange()
	type Row struct {
		Day    string
		Symbol string
		Units  string
		Price  string
	}
	// raw SQL for simplicity
	q := `
    WITH rewards_by_day AS (
        SELECT date(event_time AT TIME ZONE 'Asia/Kolkata') AS day, symbol, SUM(units::numeric) AS units
        FROM rewards WHERE user_id = ? AND event_time < ?
        GROUP BY 1,2
    ),
    last_price AS (
        SELECT DISTINCT ON (date(as_of AT TIME ZONE 'Asia/Kolkata'), symbol)
            date(as_of AT TIME ZONE 'Asia/Kolkata') AS day, symbol, price
        FROM stock_prices
        WHERE as_of < ?
        ORDER BY date(as_of AT TIME ZONE 'Asia/Kolkata'), symbol, as_of DESC
    )
    SELECT r.day::text, r.symbol, r.units::text, COALESCE(lp.price,'1000.0000')::text AS price
    FROM rewards_by_day r
    LEFT JOIN last_price lp ON lp.day=r.day AND lp.symbol=r.symbol
    ORDER BY r.day ASC, r.symbol ASC;
    `
	var rows []Row
	if err := h.db.Raw(q, userId, start, start).Scan(&rows).Error; err != nil {
		c.JSON(500, gin.H{"error": "db"})
		return
	}
	type Out struct {
		Day      string `json:"day"`
		TotalINR string `json:"totalInr"`
	}
	agg := map[string]float64{}
	for _, r := range rows {
		var u, p float64
		_, _ = fmt.Sscanf(r.Units, "%f", &u)
		_, _ = fmt.Sscanf(r.Price, "%f", &p)
		agg[r.Day] += u * p
	}
	var out []Out
	for day, v := range agg {
		out = append(out, Out{Day: day, TotalINR: fmt.Sprintf("%.4f", v)})
	}
	// stable sort by day
	sort.Slice(out, func(i, j int) bool { return out[i].Day < out[j].Day })
	c.JSON(200, out)
}

func (h *Handler) getStats(c *gin.Context) {
	userId := c.Param("userId")
	start, end := util.TodayRange()

	// today grouped shares
	type Row struct {
		Symbol string
		Units  string
	}
	var today []Row
	q1 := `SELECT symbol, SUM(units::numeric)::text AS units FROM rewards WHERE user_id = ? AND event_time >= ? AND event_time < ? GROUP BY symbol ORDER BY symbol`
	if err := h.db.Raw(q1, userId, start, end).Scan(&today).Error; err != nil {
		c.JSON(500, gin.H{"error": "db"})
		return
	}

	// holdings per symbol
	var holds []Row
	q2 := `SELECT symbol, SUM(units::numeric)::text AS units FROM rewards WHERE user_id = ? GROUP BY symbol`
	if err := h.db.Raw(q2, userId).Scan(&holds).Error; err != nil {
		c.JSON(500, gin.H{"error": "db"})
		return
	}

	// latest prices
	// compute current portfolio
	total := 0.0
	breakdown := []gin.H{}
	for _, hld := range holds {
		var u float64
		_, _ = fmt.Sscanf(hld.Units, "%f", &u)
		var sp models.StockPrice
		if err := h.db.Where("symbol = ?", hld.Symbol).Order("as_of DESC").First(&sp).Error; err != nil {
			sp.Price = "1000.0000"
		}
		var p float64
		_, _ = fmt.Sscanf(sp.Price, "%f", &p)
		v := u * p
		total += v
		breakdown = append(breakdown, gin.H{"symbol": hld.Symbol, "units": fmt.Sprintf("%.6f", u), "price": fmt.Sprintf("%.4f", p), "value": fmt.Sprintf("%.4f", v)})
	}

	c.JSON(200, gin.H{
		"todayBySymbol":       today,
		"currentPortfolioInr": fmt.Sprintf("%.4f", total),
		"breakdown":           breakdown,
	})
}

func (h *Handler) getPortfolio(c *gin.Context) {
	userId := c.Param("userId")
	type Row struct {
		Symbol string
		Units  string
	}
	var holds []Row
	q := `SELECT symbol, SUM(units::numeric)::text AS units FROM rewards WHERE user_id = ? GROUP BY symbol ORDER BY symbol`
	if err := h.db.Raw(q, userId).Scan(&holds).Error; err != nil {
		c.JSON(500, gin.H{"error": "db"})
		return
	}
	var out []gin.H
	for _, hld := range holds {
		var u float64
		_, _ = fmt.Sscanf(hld.Units, "%f", &u)
		var sp models.StockPrice
		if err := h.db.Where("symbol = ?", hld.Symbol).Order("as_of DESC").First(&sp).Error; err != nil {
			sp.Price = "1000.0000"
		}
		var p float64
		_, _ = fmt.Sscanf(sp.Price, "%f", &p)
		v := u * p
		out = append(out, gin.H{"symbol": hld.Symbol, "units": fmt.Sprintf("%.6f", u), "price": sp.Price, "inr": fmt.Sprintf("%.4f", v)})
	}
	c.JSON(200, out)
}
