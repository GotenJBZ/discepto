package db

import (
	"os"

	"gitlab.com/ranfdev/discepto/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

func Open() DB {
	// dbUrl := "postgres://discepto:passwd@localhost/disceptoDb"
	dbUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to db")
	}
	res := DB {
		db: db,
	}
	res.autoMigrate()
	return res
}
func (db *DB) autoMigrate() {
	db.db.AutoMigrate(&models.Essay{})
}

func (db *DB) CreateEssay(essay models.Essay) (*gorm.DB) {
	return db.db.Create(essay)
}

func (db *DB) ListEssays(essay models.Essay, offset int, limit int) (*gorm.DB) {
	return db.db.Offset(offset).Limit(limit)
}
