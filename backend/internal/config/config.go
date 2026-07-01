package config

import "os"

type Config struct {
	Addr           string
	DBDSN          string
	RedisAddr      string
	RedisPassword  string
	JWTSecret      string
	TokenHours     int
	SMTPHost       string
	SMTPPort       string
	SMTPUser       string
	SMTPPass       string
	DevAuthEnabled bool
	AIEnabled      bool
	AIBaseURL      string
	AIAPIKey       string
	AIModel        string
}

func Load() Config {
	return Config{
		Addr:           getEnv("APP_ADDR", ":8080"),
		DBDSN:          getEnv("DB_DSN", "health:health123@tcp(127.0.0.1:3306)/health_checkup?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:      getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		JWTSecret:      getEnv("JWT_SECRET", "dev-health-checkup-secret"),
		TokenHours:     12,
		SMTPHost:       getEnv("SMTP_HOST", "smtp.qq.com"),
		SMTPPort:       getEnv("SMTP_PORT", "587"),
		SMTPUser:       getEnv("SMTP_USER", ""),
		SMTPPass:       getEnv("SMTP_PASS", ""),
		DevAuthEnabled: getEnv("DEV_AUTH_ENABLED", "false") == "true",
		AIEnabled:      getEnv("AI_ENABLED", "true") == "true",
		AIBaseURL:      getEnv("AI_BASE_URL", "https://api.openai.com/v1"),
		AIAPIKey:       getEnv("AI_API_KEY", ""),
		AIModel:        getEnv("AI_MODEL", "gpt-4.1-mini"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
