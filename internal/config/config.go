package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName           string
	HTTPAddr          string
	DatabaseURL       string
	JWTSecret         string
	JWTExpire         time.Duration
	EncryptionKey     []byte // 32 bytes AES-256
	EngineMaxWorkers  int
	EngineTaskTimeout time.Duration
	ChromePath        string
	GinMode           string
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func Load() *Config {
	db := getenv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=agentflow port=5432 sslmode=disable")
	jwtSecret := getenv("JWT_SECRET", "dev-jwt-secret-change-me-in-production-min-32-chars!!")
	keyStr := getenv("ENCRYPTION_KEY", "")
	var encKey []byte
	if keyStr != "" {
		encKey = []byte(keyStr)
		if len(encKey) != 32 {
			// pad/truncate to 32 for dev only
			b := make([]byte, 32)
			copy(b, encKey)
			encKey = b
		}
	} else {
		encKey = []byte("0123456789abcdef0123456789abcdef") // dev only
	}
	maxw, _ := strconv.Atoi(getenv("ENGINE_MAX_WORKERS", "64"))
	taskTO, _ := time.ParseDuration(getenv("ENGINE_TASK_TIMEOUT", "30m"))
	if taskTO <= 0 {
		taskTO = 30 * time.Minute
	}
	jwtExp, _ := time.ParseDuration(getenv("JWT_EXPIRE", "24h"))
	if jwtExp <= 0 {
		jwtExp = 24 * time.Hour
	}
	return &Config{
		AppName:           getenv("APP_NAME", "AgentFlow Pro"),
		HTTPAddr:          getenv("HTTP_ADDR", ":28131"),
		DatabaseURL:       db,
		JWTSecret:         jwtSecret,
		JWTExpire:         jwtExp,
		EncryptionKey:     encKey,
		EngineMaxWorkers:  maxw,
		EngineTaskTimeout: taskTO,
		ChromePath:        getenv("CHROME_BIN", ""),
		GinMode:           strings.ToLower(getenv("GIN_MODE", "debug")),
	}
}
