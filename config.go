package main

import "os"

type Config struct {
	Port         string
	DatabasePath string
	UploadDir    string
	SessionName  string
	AdminUser    string
	AdminPass    string
}

func loadConfig() Config {
	return Config{
		Port:         getenv("PORT", "8080"),
		DatabasePath: getenv("DATABASE_PATH", "association.db"),
		UploadDir:    getenv("UPLOAD_DIR", "uploads"),
		SessionName:  "assoc_admin_session",
		AdminUser:    getenv("ADMIN_USER", "admin"),
		AdminPass:    getenv("ADMIN_PASS", "change-me-123"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
