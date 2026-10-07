// Package config reads the process configuration from the environment and
// fails fast, naming every missing or invalid variable at once.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/manikorimilli/outlet-owl/internal/gateway"
)

// minJWTSecretBytes is the shortest signing secret accepted: 32 bytes is the
// HS256 key size (phase 1 server LLD, section 7).
const minJWTSecretBytes = 32

// Config is the validated process configuration.
type Config struct {
	Port        string
	LogLevel    slog.Level
	DatabaseURL string
	// JWTSecret signs the session cookie (ADR-0007). Never log it; changing
	// it signs everyone out.
	JWTSecret []byte
	// UsersFile is the path of the users file loaded at start (HLD section 3).
	UsersFile string
	// BrandName and BrandTimezone describe the one brand per installation
	// (Q-001); GET /me returns both, and the timezone bounds the weeks.
	BrandName     string
	BrandTimezone *time.Location
	// GatewayMode is live, record or replay; unset means replay, so nothing
	// spends money unless the operator chooses it (phase 2 LLD section 7).
	GatewayMode gateway.Mode
	// OpenRouterKey is required in live and record mode. Never log it.
	OpenRouterKey string
	// RecordingsDir is where record writes and replay reads.
	RecordingsDir string
}

// Load reads and validates the environment.
func Load() (Config, error) {
	var problems []string
	c := Config{
		Port:        envOr("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   []byte(os.Getenv("JWT_SECRET")),
		UsersFile:   envOr("USERS_FILE", "users.json"),
		BrandName:   strings.TrimSpace(os.Getenv("BRAND_NAME")),
	}
	if c.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required (see .env.example)")
	}
	switch strings.ToLower(envOr("LOG_LEVEL", "info")) {
	case "debug":
		c.LogLevel = slog.LevelDebug
	case "info":
		c.LogLevel = slog.LevelInfo
	case "warn":
		c.LogLevel = slog.LevelWarn
	case "error":
		c.LogLevel = slog.LevelError
	default:
		problems = append(problems, "LOG_LEVEL must be debug, info, warn or error")
	}
	// The secret's value never appears in a message, only its length rule.
	switch n := len(c.JWTSecret); {
	case n == 0:
		problems = append(problems, "JWT_SECRET is required (generate one: openssl rand -hex 32)")
	case n < minJWTSecretBytes:
		problems = append(problems, fmt.Sprintf("JWT_SECRET must be at least %d bytes (generate one: openssl rand -hex 32)", minJWTSecretBytes))
	}
	if c.BrandName == "" {
		problems = append(problems, "BRAND_NAME is required (the brand shown after sign-in)")
	}
	tzName := envOr("BRAND_TIMEZONE", "Asia/Kolkata")
	// "Local" would make the weeks depend on the machine, and GET /me must
	// return an IANA zone name.
	if loc, err := time.LoadLocation(tzName); err != nil || tzName == "Local" {
		problems = append(problems, fmt.Sprintf("BRAND_TIMEZONE %q is not an IANA time zone (for example Asia/Kolkata)", tzName))
	} else {
		c.BrandTimezone = loc
	}
	if mode, err := gateway.ParseMode(os.Getenv("MODEL_GATEWAY_MODE")); err != nil {
		problems = append(problems, "MODEL_GATEWAY_MODE "+err.Error())
	} else {
		c.GatewayMode = mode
	}
	c.OpenRouterKey = os.Getenv("OPENROUTER_API_KEY")
	if (c.GatewayMode == gateway.Live || c.GatewayMode == gateway.Record) && c.OpenRouterKey == "" {
		problems = append(problems, "OPENROUTER_API_KEY is required when MODEL_GATEWAY_MODE is live or record")
	}
	c.RecordingsDir = envOr("MODEL_RECORDINGS_DIR", "testdata/recordings")
	if len(problems) > 0 {
		return Config{}, fmt.Errorf("config: %s", strings.Join(problems, "; "))
	}
	return c, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
