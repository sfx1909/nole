package analyser

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

//go:embed rules.toml
var defaultRules []byte

type Rule struct {
	ID          string            `toml:"id"`
	Name        string            `toml:"name"`
	Description string            `toml:"description"`
	Detect      DetectBlock       `toml:"detect"`
	Suggest     map[string]string `toml:"suggest"`
}

type DetectBlock struct {
	Packages []string `toml:"packages"`
}

type RulesFile struct {
	Rules []Rule `toml:"rules"`
}

func loadRules() ([]Rule, error) {
	var base RulesFile
	if _, err := toml.Decode(string(defaultRules), &base); err != nil {
		return nil, err
	}

	// merge user rules if present
	userPath := userRulesPath()
	if _, err := os.Stat(userPath); err == nil {
		var user RulesFile
		if _, err := toml.DecodeFile(userPath, &user); err != nil {
			return nil, err
		}
		base.Rules = append(base.Rules, user.Rules...)
	}

	return base.Rules, nil
}

func userRulesPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "nole", "rules.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nole", "rules.toml")
}
