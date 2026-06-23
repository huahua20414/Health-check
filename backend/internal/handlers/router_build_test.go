package handlers

import (
	"testing"

	"health-checkup/backend/internal/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewRouterBuildsWithoutRouteConflicts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	NewRouter(db, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}), config.Config{})
}
