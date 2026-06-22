package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	PostgresDSN        string
	TigerBeetleAddress string
	TigerBeetleCluster uint32
	JWTSecret          string
	OpenRouterAPIKey   string
	AIModel            string
	SeedDataPath       string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:               getEnv("PORT", "8080"),
		PostgresDSN:        getEnv("POSTGRES_DSN", "postgres://banca:banca_password@localhost:5432/banca_db?sslmode=disable"),
		TigerBeetleAddress: getEnv("TIGERBEETLE_ADDRESS", "localhost:3000"),
		TigerBeetleCluster: getUint32Env("TIGERBEETLE_CLUSTER_ID", 0),
		JWTSecret:          getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		OpenRouterAPIKey:   getEnv("OPENROUTER_API_KEY", ""),
		AIModel:            getEnv("AI_MODEL", "openai/gpt-4o"),
		SeedDataPath:       getEnv("SEED_DATA_PATH", "./data/datos-prueba-HNL.json"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getUint32Env(key string, fallback uint32) uint32 {
	s := os.Getenv(key)
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return fallback
	}
	return uint32(v)
}
