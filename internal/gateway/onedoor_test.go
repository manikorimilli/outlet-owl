package gateway

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One door to the model (tenet 1, AC-US-02-001-1): no Go file outside this
// package names OpenRouter's host or its chat endpoint. Config reads the key
// and hands it in, so internal/config may name the key's variable only.
func TestOnlyTheGatewayNamesOpenRouter(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", ".scratch", "web", "testdata":
				return filepath.SkipDir
			}
			if filepath.Base(filepath.Dir(path)) == "internal" && d.Name() == "gateway" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for _, needle := range []string{"openrouter.ai", "/chat/completions"} {
			if strings.Contains(text, needle) {
				t.Errorf("%s names %q: every model call goes through internal/gateway", path, needle)
			}
		}
		if strings.Contains(text, "OPENROUTER_API_KEY") && !strings.Contains(filepath.ToSlash(path), "internal/config/") {
			t.Errorf("%s reads OPENROUTER_API_KEY: only internal/config reads it and hands it to the gateway", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
