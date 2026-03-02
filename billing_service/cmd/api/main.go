package main

import (
	"log"
	"order-billing-system/billing_service/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("startup failed: %v", err)
	}
}
