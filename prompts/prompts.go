// Package prompts holds the versioned prompts the model gateway sends (Q-011,
// US-02-002, phase 2 LLD section 3). Each prompt is a directory of numbered
// version files plus a file naming the current one:
//
//	prompts/<name>/v<N>.md   a header between two "---" lines, then the text
//	prompts/<name>/current   the number of the current version
//
// Versions are never edited or deleted once used, because stored tags and
// drafts name them; a change of any kind is a new version. The current marker
// lives outside the version files, so making v2 current never touches v1.
// The real prompts arrive with their phases (tagging in 3, reply in 5), which
// add the //go:embed that feeds Parse.
package prompts

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// MaxTokensCap is the most a version may ask for (REQ-032); the gateway
// clamps to it again on every request.
const MaxTokensCap = 1000

// Version is one prompt version as the gateway sends it.
type Version struct {
	Name      string
	Number    int
	Text      string
	MaxTokens int
	// Temperature is nil when the version leaves it to the provider.
	Temperature *float64
}

// Registry is every version of every prompt and the current one of each.
type Registry struct {
	versions map[string]map[int]Version
	current  map[string]int
}

var versionFile = regexp.MustCompile(`^v([1-9][0-9]*)\.md$`)

// Parse reads every prompt directory at the root of fsys and checks the
// rules: a current file naming an existing version, numbers from 1 with no
// gap, a header with max_tokens from 1 to 1000, and text that is not blank.
// A directory named testdata is skipped.
func Parse(fsys fs.FS) (*Registry, error) {
	r := &Registry{versions: map[string]map[int]Version{}, current: map[string]int{}}
	dirs, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("prompts: read the root: %w", err)
	}
	for _, d := range dirs {
		if !d.IsDir() || d.Name() == "testdata" {
			continue
		}
		if err := r.parsePrompt(fsys, d.Name()); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) parsePrompt(fsys fs.FS, name string) error {
	entries, err := fs.ReadDir(fsys, name)
	if err != nil {
		return fmt.Errorf("prompts: read %s: %w", name, err)
	}
	versions := map[int]Version{}
	current := 0
	for _, e := range entries {
		file := path.Join(name, e.Name())
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return fmt.Errorf("prompts: read %s: %w", file, err)
		}
		if e.Name() == "current" {
			n, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil || n < 1 {
				return fmt.Errorf("prompts: %s must hold one version number", file)
			}
			current = n
			continue
		}
		m := versionFile.FindStringSubmatch(e.Name())
		if m == nil {
			return fmt.Errorf("prompts: %s is neither a version file (vN.md) nor current", file)
		}
		n, _ := strconv.Atoi(m[1]) // the pattern admits digits only
		v, err := parseVersion(name, n, string(data))
		if err != nil {
			return fmt.Errorf("prompts: %s: %w", file, err)
		}
		versions[n] = v
	}
	if len(versions) == 0 {
		return fmt.Errorf("prompts: %s has no version files", name)
	}
	numbers := make([]int, 0, len(versions))
	for n := range versions {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)
	for i, n := range numbers {
		if n != i+1 {
			return fmt.Errorf("prompts: %s has no v%d.md; versions are numbered from 1 with no gap", name, i+1)
		}
	}
	if current == 0 {
		return fmt.Errorf("prompts: %s has no current file", name)
	}
	if _, ok := versions[current]; !ok {
		return fmt.Errorf("prompts: %s/current names v%d, which does not exist", name, current)
	}
	r.versions[name] = versions
	r.current[name] = current
	return nil
}

func parseVersion(name string, n int, data string) (Version, error) {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	rest, ok := strings.CutPrefix(data, "---\n")
	if !ok {
		return Version{}, errors.New(`the file must start with a "---" header`)
	}
	header, text, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return Version{}, errors.New(`the header has no closing "---" line`)
	}
	v := Version{Name: name, Number: n, Text: strings.TrimSpace(text)}
	for _, line := range strings.Split(header, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Version{}, fmt.Errorf("header line %q is not key: value", line)
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "max_tokens":
			mt, err := strconv.Atoi(value)
			if err != nil || mt < 1 || mt > MaxTokensCap {
				return Version{}, fmt.Errorf("max_tokens must be from 1 to %d", MaxTokensCap)
			}
			v.MaxTokens = mt
		case "temperature":
			t, err := strconv.ParseFloat(value, 64)
			if err != nil || t < 0 || t > 2 {
				return Version{}, errors.New("temperature must be a number from 0 to 2")
			}
			v.Temperature = &t
		default:
			return Version{}, fmt.Errorf("unknown header key %q", strings.TrimSpace(key))
		}
	}
	if v.MaxTokens == 0 {
		return Version{}, errors.New("the header must set max_tokens")
	}
	if v.Text == "" {
		return Version{}, errors.New("the prompt text is blank")
	}
	return v, nil
}

// Current returns the current version of a prompt.
func (r *Registry) Current(name string) (Version, error) {
	n, ok := r.current[name]
	if !ok {
		return Version{}, fmt.Errorf("prompts: no prompt named %q", name)
	}
	return r.versions[name][n], nil
}

// Get returns one version of a prompt, for reading what a stored tag or
// draft was made with.
func (r *Registry) Get(name string, n int) (Version, error) {
	v, ok := r.versions[name][n]
	if !ok {
		return Version{}, fmt.Errorf("prompts: no version %d of %q", n, name)
	}
	return v, nil
}
