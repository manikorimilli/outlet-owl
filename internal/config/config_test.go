package config

import (
	"log/slog"
	"strings"
	"testing"
)

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
