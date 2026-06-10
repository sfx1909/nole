package flake

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sfx1909/nole/internal/config"
)

type Context struct {
	FlakePath  string
	ConfigName string
}

var systemNameRe = regexp.MustCompile(`nixos-system-(.+?)-\d+\.\d+`)

// Detect finds the nearest flake.nix and the current NixOS config name.
func Detect() (*Context, error) {
	spec, err := findFlake()
	if err != nil {
		return nil, err
	}

	flakePath, configName, _ := strings.Cut(spec, "#")

	if configName == "" {
		configName, err = resolveConfigName(flakePath)
		if err != nil {
			return nil, err
		}
	}

	return &Context{
		FlakePath:  flakePath,
		ConfigName: configName,
	}, nil
}

func resolveConfigName(flakePath string) (string, error) {
	// list available nixosConfigurations in the flake
	out, err := exec.Command("nix", "eval", "--json",
		flakePath+"#nixosConfigurations",
		"--apply", "builtins.attrNames",
	).Output()
	if err != nil {
		// fall back to boot.json parsing if eval fails
		return currentConfigName()
	}

	var names []string
	if err := json.Unmarshal(out, &names); err != nil || len(names) == 0 {
		return currentConfigName()
	}

	if len(names) == 1 {
		return names[0], nil
	}

	// try to match by hostname first
	hostname, _ := os.Hostname()
	for _, name := range names {
		if name == hostname {
			return name, nil
		}
	}

	// try to match against the system name from boot.json
	sysName, err := currentConfigName()
	if err == nil {
		for _, name := range names {
			if name == sysName {
				return name, nil
			}
		}
	}

	// multiple configs, none matched — return them all in the error so user knows
	return "", fmt.Errorf("multiple nixosConfigurations found %v — set flake in ~/.config/nole/config.toml or use NOLE_FLAKE=path#name", names)
}

func findFlake() (string, error) {
	// prefer explicit env var
	if p := os.Getenv("NOLE_FLAKE"); p != "" {
		return p, nil
	}

	// then config file
	cfg, err := config.Load()
	if err == nil && cfg.Flake != "" {
		return cfg.Flake, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "flake.nix")
		if _, err := os.Stat(candidate); err == nil {
			if hasNixosConfigurations(candidate) {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no flake.nix with nixosConfigurations found in current directory or any parent")
}

func hasNixosConfigurations(flakeFile string) bool {
	data, err := os.ReadFile(flakeFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "nixosConfigurations")
}

func currentConfigName() (string, error) {
	data, err := os.ReadFile("/run/current-system/boot.json")
	if err != nil {
		return "", fmt.Errorf("could not read system boot.json: %w", err)
	}

	var boot struct {
		V1 struct {
			Toplevel string `json:"toplevel"`
		} `json:"org.nixos.bootspec.v1"`
	}

	if err := json.Unmarshal(data, &boot); err != nil {
		return "", fmt.Errorf("could not parse boot.json: %w", err)
	}

	base := filepath.Base(boot.V1.Toplevel)
	// strip the nix store hash prefix (hash-nixos-system-name-version)
	parts := strings.SplitN(base, "-", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected toplevel format: %s", base)
	}

	matches := systemNameRe.FindStringSubmatch(parts[1])
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract config name from: %s", parts[1])
	}

	return matches[1], nil
}
