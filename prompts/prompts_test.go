package prompts

import (
	"strings"
	"testing"
	"testing/fstest"
)

func file(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

const v1 = "---\nmax_tokens: 1000\ntemperature: 0\n---\nTag each review.\n"
const v2 = "---\nmax_tokens: 800\n---\nTag each review, briefly.\n"

func TestParse_LoadsVersionsAndTheCurrentOne(t *testing.T) {
	r, err := Parse(fstest.MapFS{
		"tagging/v1.md":   file(v1),
		"tagging/v2.md":   file(v2),
		"tagging/current": file("2\n"),
		"testdata/x.md":   file("ignored"),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	cur, err := r.Current("tagging")
	if err != nil || cur.Number != 2 || cur.MaxTokens != 800 || cur.Text != "Tag each review, briefly." {
		t.Fatalf("Current = %+v, %v; want v2", cur, err)
	}
	old, err := r.Get("tagging", 1)
	if err != nil || old.Number != 1 || old.Text != "Tag each review." {
		t.Fatalf("Get(1) = %+v, %v; want v1 kept", old, err)
	}
	if _, err := r.Current("reply"); err == nil {
		t.Fatal("Current of an unknown prompt must fail")
	}
}

func TestParse_ReadsTemperatureOnlyWhenSet(t *testing.T) {
	r, err := Parse(fstest.MapFS{"a/v1.md": file(v1), "a/v2.md": file(v2), "a/current": file("1")})
	if err != nil {
		t.Fatal(err)
	}
	one, _ := r.Get("a", 1)
	two, _ := r.Get("a", 2)
	if one.Temperature == nil || *one.Temperature != 0 || two.Temperature != nil {
		t.Fatalf("temperature v1 = %v, v2 = %v; want 0 and unset", one.Temperature, two.Temperature)
	}
}

func TestParse_Refuses(t *testing.T) {
	cases := []struct {
		name string
		fs   fstest.MapFS
		want string
	}{
		{"CurrentNamingAMissingVersion", fstest.MapFS{"a/v1.md": file(v1), "a/current": file("2")}, "v2, which does not exist"},
		{"AGapInVersionNumbers", fstest.MapFS{"a/v1.md": file(v1), "a/v3.md": file(v2), "a/current": file("1")}, "no v2.md"},
		{"MaxTokensAbove1000", fstest.MapFS{"a/v1.md": file("---\nmax_tokens: 4000\n---\nx"), "a/current": file("1")}, "max_tokens must be from 1 to 1000"},
		{"NaNTemperature", fstest.MapFS{"a/v1.md": file("---\nmax_tokens: 10\ntemperature: NaN\n---\nx"), "a/current": file("1")}, "temperature must be a number from 0 to 2"},
		{"MissingMaxTokens", fstest.MapFS{"a/v1.md": file("---\ntemperature: 0\n---\nx"), "a/current": file("1")}, "must set max_tokens"},
		{"AnUnknownHeaderKey", fstest.MapFS{"a/v1.md": file("---\nmax_tokens: 10\nmodel: x\n---\nx"), "a/current": file("1")}, `unknown header key "model"`},
		{"BlankText", fstest.MapFS{"a/v1.md": file("---\nmax_tokens: 10\n---\n  \n"), "a/current": file("1")}, "blank"},
		{"NoHeader", fstest.MapFS{"a/v1.md": file("Tag it."), "a/current": file("1")}, "must start"},
		{"NoCurrentFile", fstest.MapFS{"a/v1.md": file(v1)}, "no current file"},
		{"AStrayFile", fstest.MapFS{"a/v1.md": file(v1), "a/current": file("1"), "a/notes.txt": file("x")}, "neither a version file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.fs)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Parse err = %v, want one containing %q", err, tc.want)
			}
		})
	}
}
