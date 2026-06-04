package config

import "os"

type Config struct {
	Addr  string
	DBDSN string
}

func Load() Config {
	return Config{
		Addr:  getEnv("APP_ADDR", ":8080"),
		DBDSN: getEnv("DB_DSN", "health:health123@tcp(127.0.0.1:3306)/health_checkup?charset=utf8mb4&parseTime=True&loc=Local"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
