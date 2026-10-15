package models

func All() []interface{} {
	return []interface{}{
		&Budget{},
		&BudgetSection{},
		&BudgetSectionItem{},
	}
}
