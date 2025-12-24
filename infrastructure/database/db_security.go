package database

import (
	"errors"
	"log"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// pola yang dianggap berbahaya
var forbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i);`),                     // pemisah perintah
	regexp.MustCompile(`(?i)--`),                    // komentar SQL
	regexp.MustCompile(`(?i)/\*.*\*/`),              // komentar block
	regexp.MustCompile(`(?i)\bdrop\b`),              // DROP
	regexp.MustCompile(`(?i)\balter\b`),             // ALTER
	regexp.MustCompile(`(?i)\btruncate\b`),          // TRUNCATE
	regexp.MustCompile(`(?i)\bdelete\b`),            // DELETE tanpa where
	regexp.MustCompile(`(?i)\bunion\b.*\bselect\b`), // UNION SELECT
	regexp.MustCompile(`(?i)<script>`),              // XSS dalam query
}

// cek jika query raw mengandung pola terlarang
func isQueryDangerous(sql string) bool {
	clean := strings.TrimSpace(sql)
	for _, pattern := range forbiddenPatterns {
		if pattern.MatchString(clean) {
			return true
		}
	}
	return false
}

// callback sebelum query dieksekusi
func registerQueryProtector(db *gorm.DB) {
	callback := db.Callback()

	// semua Raw SQL dan Exec SQL akan lewat sini
	callback.Raw().Before("gorm:raw").Register("security:check_raw_sql", func(db *gorm.DB) {
		if db.Statement.SQL.String() != "" {
			sql := db.Statement.SQL.String()

			if isQueryDangerous(sql) {
				log.Printf("🚨 DIBLOKIR! SQL mencurigakan: %s\n", sql)
				db.AddError(errors.New("query mencurigakan diblokir demi keamanan"))
			}
		}
	})

	callback.Update().Before("gorm:update").Register("security:check_update", func(db *gorm.DB) {
		sql := db.Statement.SQL.String()
		if isQueryDangerous(sql) {
			log.Printf("🚨 DIBLOKIR! UPDATE mencurigakan: %s\n", sql)
			db.AddError(errors.New("update mencurigakan diblokir demi keamanan"))
		}
	})

	callback.Query().Before("gorm:query").Register("security:check_query", func(db *gorm.DB) {
		sql := db.Statement.SQL.String()
		if isQueryDangerous(sql) {
			log.Printf("🚨 DIBLOKIR! QUERY mencurigakan: %s\n", sql)
			db.AddError(errors.New("query mencurigakan diblokir demi keamanan"))
		}
	})
}
