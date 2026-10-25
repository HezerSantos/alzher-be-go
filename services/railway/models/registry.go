package models

import (
	"log"

	"gorm.io/gorm"
)

func All() []interface{} {
	return []interface{}{
		&Budget{},
		&BudgetSection{},
		&BudgetSectionItem{},
	}
}

func Reset(db *gorm.DB) {
	if err := db.Exec(`
    DROP SCHEMA public CASCADE;
    CREATE SCHEMA public;
`).Error; err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(
		&User{},
		&Statements{},
		&Budget{},
		&BudgetSection{},
		&BudgetSectionItem{},
		&Transaction{},
	); err != nil {
		log.Fatal(err)
	}
}
