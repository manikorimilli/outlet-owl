package prompts

import "embed"

// files holds the prompt versions built into the binary (Q-011). reply/
// joins with build phase 5.
//
//go:embed tagging
var files embed.FS

// Load parses the prompts built into the binary.
func Load() (*Registry, error) { return Parse(files) }
