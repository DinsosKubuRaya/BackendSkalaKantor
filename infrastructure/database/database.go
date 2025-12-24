package database

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	_ = godotenv.Load()

	mysqlURL := os.Getenv("MYSQL_URL")
	if mysqlURL == "" {
		log.Fatal("❌ MYSQL_URL tidak ditemukan")
	}

	// mysql://user:pass@host:port/dbname
	mysqlURL = strings.TrimPrefix(mysqlURL, "mysql://")

	// tambahkan parameter wajib gorm
	dsn := mysqlURL + "?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Gagal koneksi database:", err)
	}

	registerQueryProtector(db)

	DB = db
	log.Println("✅ Database connected using MYSQL_URL")
}
