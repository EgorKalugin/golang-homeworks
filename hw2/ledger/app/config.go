package app

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"ledger/internal/cache"
)

type Config struct {
	DSN   string
	Redis cache.Config
}

func ConfigFromEnv() (Config, error) {
	redisCfg, err := cache.ConfigFromEnv()
	if err != nil {
		return Config{}, fmt.Errorf("redis config: %w", err)
	}
	return Config{DSN: dsnFromEnv(), Redis: redisCfg}, nil
}

func dsnFromEnv() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(getenv("DB_USER", "postgres"), getenv("DB_PASS", "postgres")),
		Host:     net.JoinHostPort(getenv("DB_HOST", "localhost"), getenv("DB_PORT", "5432")),
		Path:     "/" + getenv("DB_NAME", "cashapp"),
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
