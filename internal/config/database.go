package config

import (
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(goEnv, dbUrl string) (*gorm.DB, error) {
	var err error

	if goEnv == "development" && dbUrl == "" {
		DB, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	} else if strings.HasPrefix(dbUrl, "postgres") {
		DB, err = gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	} else {
		DB, err = gorm.Open(sqlite.Open(dbUrl), &gorm.Config{})
	}

	if err != nil {
		return nil, err
	}

	return DB, nil
}

func GetDB() *gorm.DB {
	return DB
}
