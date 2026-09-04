package railway

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() error {
	var DATABASE_URL = os.Getenv("DATABASE_URL")
	if DATABASE_URL == "" {
		return fmt.Errorf("DATA BASE URL NOT CONFIGURED")
	}

	db, err := gorm.Open(postgres.Open(DATABASE_URL))

	if err != nil {
		return err
	}

	DB = db
	return nil
}
