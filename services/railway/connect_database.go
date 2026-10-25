package railway

import (
	"fmt"
	"log"
	"os"

	"github.com/HezerSantos/alzher-api/services/railway/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() error {
	var DATABASE_URL = os.Getenv("DATABASE_URL")
	var MIGRATE_DATABASE = os.Getenv("MIGRATE_DATABASE")
	var RESET_DATABASE = os.Getenv("RESET_DATABASE")
	if DATABASE_URL == "" {
		return fmt.Errorf("DATA BASE URL NOT CONFIGURED")
	}

	db, err := gorm.Open(postgres.Open(DATABASE_URL))

	if err != nil {
		return err
	}

	if MIGRATE_DATABASE == "true" {
		log.Println("Database migration start...")
		if err := db.AutoMigrate(models.All()...); err != nil {
			log.Fatal(err)
		}
		log.Println("Database migration completed")
	}

	if RESET_DATABASE == "true" {
		log.Println("Resetting Database...")
		models.Reset(db)
		log.Println("Database migration completed")
	}

	DB = db
	return nil
}
