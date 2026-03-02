package database

import (
	"fmt"
	"log"
	"time"

	m "order-billing-system/order_service/internal/domain/model"

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

	if err := db.AutoMigrate(&m.Order{}); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	log.Println("database connected and migrated successfully")
	return db, nil
}
