package prompts

import "embed"

// files holds the prompt versions built into the binary (Q-011): tagging
// (phase 3) and reply (phase 5).
//
//go:embed tagging reply
var files embed.FS

// Load parses the prompts built into the binary.
func Load() (*Registry, error) { return Parse(files) }
