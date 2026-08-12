package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	SessionTTL    time.Duration
	EISBaseURL    string
	EISCADir      string
	EISTLSInsecure bool
	CoreURL       string
	SeedDemoUser  bool
}

func FromEnv() Config {
	ttlHours := envInt("SESSION_TTL_HOURS", 72)
	return Config{
		HTTPAddr:        env("HTTP_ADDR", ":8091"),
		DatabaseURL:     env("DATABASE_URL", "postgres://zakupki:zakupki@localhost:5432/zakupki_search?sslmode=disable"),
		SessionTTL:      time.Duration(ttlHours) * time.Hour,
		EISBaseURL:      env("EIS_BASE_URL", "https://zakupki.gov.ru"),
		EISCADir:       env("EIS_CA_DIR", "certs"),
		// Default true: Go cannot always verify Minцифры chains (wrong Sub gen / GOST).
		// Set EIS_TLS_INSECURE=false to enforce embedded CA verify.
		EISTLSInsecure: env("EIS_TLS_INSECURE", "true") == "true",
		CoreURL:        env("CORE_URL", ""),
		SeedDemoUser:   env("SEED_DEMO_USER", "true") == "true",
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
