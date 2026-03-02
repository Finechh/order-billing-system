package database

import (
	"fmt"
	"log"
	"order-billing-system/billing_service/internal/billing_domain/events"
	i "order-billing-system/billing_service/internal/billing_domain/invoice"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	maxRetries    = 10
	retryInterval = 3 * time.Second
)

func InitDB(dsn string) (*gorm.DB, error) {
	var (
		db  *gorm.DB
		err error
	)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Error),
		})
		if err == nil {
			break
		}
		log.Printf("[database] attempt %d/%d failed: %v — retrying in %s",
			attempt, maxRetries, err, retryInterval)
		time.Sleep(retryInterval)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("database connected and migrated successfully")
	return db, nil
}

func runMigrations(db *gorm.DB) error {
	db.Exec(`ALTER TABLE invoices DROP CONSTRAINT IF EXISTS uni_invoices_order_id`)
	db.Exec(`ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_order_id_key`)

	return db.AutoMigrate(&i.Invoice{}, &events.ProcessedEvent{})
}
