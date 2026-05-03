package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port string
	Host string

	// Fastly Fanout / GRIP
	FanoutEnabled bool
	GRIPURL       string
	GRIPKey       string
	FanoutDomain  string

	// Chat limits
	MaxRooms           int
	MaxMessagesPerRoom int
	MaxUsernameLength  int
	MaxMessageLength   int
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
		Host: getEnv("HOST", "0.0.0.0"),

		FanoutEnabled: getEnvBool("FASTLY_FANOUT_ENABLED", false),
		GRIPURL:       getEnv("FASTLY_GRIP_URL", ""),
		GRIPKey:       getEnv("FASTLY_GRIP_KEY", ""),
		FanoutDomain:  getEnv("FASTLY_FANOUT_DOMAIN", ""),

		MaxRooms:           getEnvInt("MAX_ROOMS", 100),
		MaxMessagesPerRoom: getEnvInt("MAX_MESSAGES_PER_ROOM", 500),
		MaxUsernameLength:  getEnvInt("MAX_USERNAME_LENGTH", 32),
		MaxMessageLength:   getEnvInt("MAX_MESSAGE_LENGTH", 2000),
	}
}

// Addr returns the combined host:port listen address
func (c *Config) Addr() string {
	return c.Host + ":" + c.Port
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		lower := strings.ToLower(val)
		return lower == "true" || lower == "1" || lower == "yes"
	}
	return fallback
}
