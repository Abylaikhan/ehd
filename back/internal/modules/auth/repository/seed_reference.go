package repository

import "gorm.io/gorm"

// SeedReference идемпотентно наполняет справочники регионов и подразделений
// демо-значениями sandbox (совпадают с кодами в ClickHouse demo_transactions).
func SeedReference(db *gorm.DB) error {
	regions := []RegionModel{
		{Code: "AST", NameRu: "Астана", NameKk: "Астана", Status: "active"},
		{Code: "ALA", NameRu: "Алматы", NameKk: "Алматы", Status: "active"},
		{Code: "SHY", NameRu: "Шымкент", NameKk: "Шымкент", Status: "active"},
	}
	departments := []DepartmentModel{
		{Code: "D01", NameRu: "Подразделение 01", NameKk: "Бөлімше 01", Status: "active"},
		{Code: "D02", NameRu: "Подразделение 02", NameKk: "Бөлімше 02", Status: "active"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for i := range regions {
			r := regions[i]
			if err := tx.Where(RegionModel{Code: r.Code}).
				Attrs(RegionModel{NameRu: r.NameRu, NameKk: r.NameKk, Status: r.Status}).
				FirstOrCreate(&regions[i]).Error; err != nil {
				return err
			}
		}
		for i := range departments {
			d := departments[i]
			if err := tx.Where(DepartmentModel{Code: d.Code}).
				Attrs(DepartmentModel{NameRu: d.NameRu, NameKk: d.NameKk, Status: d.Status}).
				FirstOrCreate(&departments[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
