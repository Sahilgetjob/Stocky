package main

import (
	"log"
	"os"

	"github.com/Sahilgetjob/stocky-backend/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
	_ = os.Stdout.Sync()
}
