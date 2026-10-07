package config

import (
	"log/slog"
	"strings"
	"testing"
)

const validSecret = "0123456789abcdef0123456789abcdef" // 32 bytes, test only

// setValidEnv sets every required variable to a valid value; a test then
// overrides the one it is about. t.Setenv restores them after the test.
func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("USERS_FILE", "")
	t.Setenv("BRAND_NAME", "Neem Tree Kitchens")
	t.Setenv("BRAND_TIMEZONE", "")
}

func TestLoad(t *testing.T) {
	cases := []struct {
		name    string
		dbURL   string
		level   string
		want    slog.Level
		wantErr string
	}{
		{name: "default level is info", dbURL: "postgres://x", want: slog.LevelInfo},
		{name: "debug parses", dbURL: "postgres://x", level: "debug", want: slog.LevelDebug},
		{name: "unknown level is an error", dbURL: "postgres://x", level: "loud", wantErr: "LOG_LEVEL"},
		{name: "missing database url is an error", wantErr: "DATABASE_URL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("DATABASE_URL", tc.dbURL)
			t.Setenv("LOG_LEVEL", tc.level)

			got, err := Load()

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want an error naming %s, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.LogLevel != tc.want {
				t.Fatalf("level: got %v want %v", got.LogLevel, tc.want)
			}
		})
	}
}

func TestLoad_RequiresJWTSecretOfAtLeast32Bytes(t *testing.T) {
	cases := []struct {
		name    string
		secret  string
		wantErr string
	}{
		{name: "missing", secret: "", wantErr: "JWT_SECRET is required"},
		{name: "31 bytes", secret: validSecret[:31], wantErr: "JWT_SECRET must be at least 32 bytes"},
		{name: "32 bytes", secret: validSecret},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("JWT_SECRET", tc.secret)

			got, err := Load()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(got.JWTSecret) != tc.secret {
					t.Fatalf("JWTSecret not loaded")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want an error containing %q, got %v", tc.wantErr, err)
			}
			if tc.secret != "" && strings.Contains(err.Error(), tc.secret) {
				t.Fatalf("the error leaks the secret: %v", err)
			}
		})
	}
}

func TestLoad_RequiresBrandName(t *testing.T) {
	for _, value := range []string{"", "   "} {
		t.Run("value "+strings.ReplaceAll(value, " ", "_"), func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("BRAND_NAME", value)

			_, err := Load()

			if err == nil || !strings.Contains(err.Error(), "BRAND_NAME is required") {
				t.Fatalf("want an error naming BRAND_NAME, got %v", err)
			}
		})
	}
}

func TestLoad_DefaultsUsersFileAndTimezone(t *testing.T) {
	setValidEnv(t)

	got, err := Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UsersFile != "users.json" {
		t.Fatalf("UsersFile: got %q want users.json", got.UsersFile)
	}
	if got.BrandTimezone == nil || got.BrandTimezone.String() != "Asia/Kolkata" {
		t.Fatalf("BrandTimezone: got %v want Asia/Kolkata", got.BrandTimezone)
	}
	if got.BrandName != "Neem Tree Kitchens" {
		t.Fatalf("BrandName: got %q", got.BrandName)
	}
}

func TestLoad_RejectsUnknownTimezone(t *testing.T) {
	cases := []struct {
		name    string
		tz      string
		wantErr bool
	}{
		{name: "unknown zone", tz: "Mars/Olympus", wantErr: true},
		{name: "machine local zone", tz: "Local", wantErr: true},
		{name: "another IANA zone", tz: "Europe/London", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("BRAND_TIMEZONE", tc.tz)

			got, err := Load()

			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "BRAND_TIMEZONE") {
					t.Fatalf("want an error naming BRAND_TIMEZONE, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.BrandTimezone.String() != tc.tz {
				t.Fatalf("BrandTimezone: got %v want %s", got.BrandTimezone, tc.tz)
			}
		})
	}
}

func TestLoad_ReportsEveryProblemAtOnce(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("BRAND_NAME", "")

	_, err := Load()

	for _, name := range []string{"DATABASE_URL", "JWT_SECRET", "BRAND_NAME"} {
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("want an error naming %s, got %v", name, err)
		}
	}
}
