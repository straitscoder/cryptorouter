package model

import (
	"database/sql"
	"log"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	DB *sql.DB
	db *gorm.DB
)

func init() {
	dsn := filepath.Join(database.DB.DataPath, "cryptorouter.db?cache=shared&mode=rwc")
	dialector := sqlite.Open(dsn)

	dbs, err := gorm.Open(dialector, &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal(err)
	}

	DB, err := dbs.DB()
	if err != nil {
		log.Fatal(err)
	}

	DB.SetMaxOpenConns(2)
	if err := dbs.AutoMigrate(&Order{}, &Trade{}); err != nil {
		log.Fatal(err)
	}

	db = dbs
}

func CloseDB() {
	defer DB.Close()
}
