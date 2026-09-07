package models

import (
	"time"

	"github.com/google/uuid"
)

type Statements struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	StatementID string    `gorm:"column:statementId"`
	UserID      uuid.UUID `gorm:"column:userId;type:uuid"`
	User        User      `gorm:"foreignKey:UserID;references:ID"`
}

type User struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	Email     string    `gorm:"column:email"`
	Password  string    `gorm:"column:password"`
	CreatedAt time.Time `gorm:"column:createdAt"`
}

type Transaction struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	Category    string    `gorm:"column:category;type:text"`
	Description string    `gorm:"column:description;type:text"`
	Amount      float64   `gorm:"column:amount;type:double precision"`
	Day         int       `gorm:"column:day;type:integer"`
	Month       string    `gorm:"column:month;type:text"`
	Year        int       `gorm:"column:year;type:integer"`
	UserID      uuid.UUID `gorm:"column:userId;type:uuid"`
	User        User      `json:"-" gorm:"foreignKey:UserID;references:ID"`
}

func (User) TableName() string {
	return "User"
}

func (Transaction) TableName() string {
	return "Transaction"
}

func (Statements) TableName() string {
	return "Statements"
}
