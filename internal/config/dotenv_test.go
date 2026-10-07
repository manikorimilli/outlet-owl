package config

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var dotenvLine = regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)=(.*)$`)

// make dev reads .env as shell (set -a; . ./.env), so every value in
// .env.example must survive that: an unquoted value with a space runs the
// rest of the line as a command and leaves the variable unset, and the server
// then refuses to start.
func TestEnvExampleLoadsLikeMakeDev(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("..", "..", ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open .env.example: %v", err)
	}
	defer func() { _ = f.Close() }()

	want := map[string]string{}
	var keys []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := dotenvLine.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		keys = append(keys, m[1])
		want[m[1]] = strings.TrimSuffix(strings.TrimPrefix(m[2], `"`), `"`)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal(".env.example names no variables")
	}

	script := `set -a; . "$0"; set +a; for k in "$@"; do printf '%s=%s\n' "$k" "${!k-<unset>}"; done`
	cmd := exec.Command("bash", append([]string{"-c", script, path}, keys...)...)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("source .env.example: %v\n%s", err, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Errorf("sourcing .env.example printed errors:\n%s", stderr.String())
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		k, got, _ := strings.Cut(line, "=")
		if got != want[k] {
			t.Errorf("%s = %q after sourcing, want %q (quote values that hold spaces)", k, got, want[k])
		}
	}
}

// makeEnv is the environment for a nested make: without the variables an
// outer make or CI may set, so only .env and the command line decide.
func makeEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		switch k, _, _ := strings.Cut(kv, "="); k {
		case "DATABASE_URL", "POSTGRES_PORT", "MAKEFLAGS", "MFLAGS", "MAKELEVEL":
		default:
			env = append(env, kv)
		}
	}
	return env
}

// migrateCommand prints, without running it, the goose command make migrate
// would run in dir.
func migrateCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	makefile, err := filepath.Abs(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", append([]string{"-n", "--no-print-directory", "-f", makefile, "-C", dir, "migrate"}, args...)...)
	cmd.Env = makeEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n migrate: %v\n%s", err, out)
	}
	return string(out)
}

// make migrate must reach the database the server uses. make db puts
// PostgreSQL on the port .env names (docker-compose.yml) and make dev reads
// DATABASE_URL from .env, but make does not read .env itself, so make migrate
// used to go to localhost:5432 whatever .env said.
func TestMakeMigrateUsesTheDatabaseURLFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	const fromFile = "postgres://postgres:postgres@localhost:55999/outlet_owl?sslmode=disable"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("BRAND_NAME=\"Neem Tree Kitchens\"\nDATABASE_URL="+fromFile+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := migrateCommand(t, dir); !strings.Contains(got, `"`+fromFile+`" up`) {
		t.Errorf("make migrate with .env runs:\n%s\nwant the DATABASE_URL from .env", got)
	}
	const fromCommandLine = "postgres://postgres:postgres@localhost:56000/other?sslmode=disable"
	if got := migrateCommand(t, dir, "DATABASE_URL="+fromCommandLine); !strings.Contains(got, `"`+fromCommandLine+`" up`) {
		t.Errorf("make migrate DATABASE_URL=... runs:\n%s\nwant the command line to win over .env", got)
	}
}

func TestMakeMigrateWithoutDotEnvUsesPostgresPort(t *testing.T) {
	dir := t.TempDir()

	if got := migrateCommand(t, dir); !strings.Contains(got, "@localhost:5432/outlet_owl") {
		t.Errorf("make migrate without .env runs:\n%s\nwant the default port 5432", got)
	}
	if got := migrateCommand(t, dir, "POSTGRES_PORT=55432"); !strings.Contains(got, "@localhost:55432/outlet_owl") {
		t.Errorf("make migrate POSTGRES_PORT=55432 runs:\n%s\nwant port 55432", got)
	}
}
