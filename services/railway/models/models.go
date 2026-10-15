package models

import (
	"time"

	"github.com/google/uuid"
)

type Statements struct {
	ID          uuid.UUID `gorm:"column:id;type:text;primaryKey"`
	StatementID string    `gorm:"column:statementId"`
	UserID      uuid.UUID `gorm:"column:userId;type:text"`
	User        User      `gorm:"foreignKey:UserID;references:ID"`
}

type User struct {
	ID        uuid.UUID `gorm:"column:id;type:text;primaryKey"`
	Email     string    `gorm:"column:email"`
	Password  string    `json:"-" gorm:"column:password"`
	CreatedAt time.Time `gorm:"column:createdAt"`
}

type Transaction struct {
	ID          uuid.UUID `gorm:"column:id;type:text;primaryKey"`
	Category    string    `gorm:"column:category;type:text"`
	Description string    `gorm:"column:description;type:text"`
	Amount      float64   `gorm:"column:amount;type:double precision"`
	Day         int       `gorm:"column:day;type:integer"`
	Month       string    `gorm:"column:month;type:text"`
	Year        int       `gorm:"column:year;type:integer"`
	UserID      uuid.UUID `gorm:"column:userId;type:text"`
	User        User      `json:"-" gorm:"foreignKey:UserID;references:ID"`
	StatementID string    `json:"statementId" gorm:"column:statementHash"`
}

type Budget struct {
	ID             uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	Name           string          `json:"name" gorm:"not null"`
	UserID         uuid.UUID       `json:"userId" gorm:"type:uuid;not null"`
	User           User            `json:"user,omitempty" gorm:"foreignKey:UserID"`
	GrossIncome    float64         `json:"grossIncome" gorm:"not null"`
	BudgetSections []BudgetSection `json:"budgetSections" gorm:"foreignKey:BudgetID"`
	IsActive       bool            `json:"isActive" gorm:"not null"`
}

type BudgetSection struct {
	ID                 uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey"`
	Name               string              `json:"name" gorm:"not null"`
	BudgetID           uuid.UUID           `json:"budgetId" gorm:"type:uuid;not null;uniqueIndex:idx_one_deduction_per_budget,where:is_deductions = true"`
	Budget             Budget              `json:"-" gorm:"foreignKey:BudgetID"`
	IsDeductions       bool                `json:"isDeductions" gorm:"not null;default:false"`
	BudgetSectionItems []BudgetSectionItem `json:"budgetSectionItems" gorm:"foreignKey:BudgetSectionID"`
}

type BudgetSectionItem struct {
	ID              uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey"`
	BudgetSectionID uuid.UUID     `json:"budgetSectionId" gorm:"type:uuid;not null"`
	BudgetSection   BudgetSection `json:"-" gorm:"foreignKey:BudgetSectionID"`
	CategoryName    string        `json:"categoryName" gorm:"not null"`
	Amount          float64       `json:"amount" gorm:"not null"`
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

func (Budget) TableName() string {
	return "Budget"
}

func (BudgetSection) TableName() string {
	return "BudgetSection"
}

func (BudgetSectionItem) TableName() string {
	return "BudgetSectionItem"
}
