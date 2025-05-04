package persistence

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func ConnectDb() *gorm.DB {
	if db != nil {
		return db
	}

	dsn := "host=localhost port=5432 user=bacon password=b4c0n dbname=bacon-db sslmode=disable"

	newDb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	db = newDb

	return db
}
