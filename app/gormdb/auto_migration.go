package gormdb

import (
	"log"

	"sonartest_cart/app/internal"

	"gorm.io/gorm"
)

func Automigration(db *gorm.DB) error {
	if err := db.AutoMigrate(&internal.Userdetail{}); err != nil {
		log.Fatalf("Migration error for user:%v", err)
	}
	log.Println("Migration success")
	return nil
}
