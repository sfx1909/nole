package purger

import "regexp"

// targets are directory basenames considered safe-to-delete build artifacts.
var targets = map[string]bool{
	"node_modules":  true,
	"target":        true,
	"dist":          true,
	"build":         true,
	".direnv":       true,
	"__pycache__":   true,
	".venv":         true,
	".pytest_cache": true,
	".mypy_cache":   true,
	".next":         true,
	".nuxt":         true,
}

// resultLinkRe matches nix build output symlinks, e.g. result, result-bin, result-dev.
var resultLinkRe = regexp.MustCompile(`^result(-.*)?$`)
