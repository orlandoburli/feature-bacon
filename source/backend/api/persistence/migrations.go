package persistence

import "fmt"

func AddExtraFunctions() error {

	db := ConnectDb()

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	return nil
}
