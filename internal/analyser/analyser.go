package analyser

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/sfx1909/nole/internal/flake"
)

type Match struct {
	Rule        Rule
	Suggestions map[string]string
}

func Run(apply bool) error {
	ctx, err := flake.Detect()
	if err != nil {
		return err
	}

	rules, err := loadRules()
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = color.New(color.Faint).Sprint("  Evaluating config")
	s.Start()

	packages, err := evalPackages(ctx)
	s.Stop()
	if err != nil {
		return fmt.Errorf("failed to evaluate packages: %w", err)
	}

	matches := match(rules, packages)

	if len(matches) == 0 {
		fmt.Println(color.GreenString("  󰄬  No optimisations found"))
		fmt.Println()
		return nil
	}

	printMatches(matches)

	if !apply {
		fmt.Printf("  %s Run with %s to generate modules\n\n",
			color.New(color.Faint).Sprint("→"),
			color.CyanString("--apply"),
		)
		return nil
	}

	if err := writeModules(matches, ctx.FlakePath); err != nil {
		return fmt.Errorf("failed to write optimisation modules: %w", err)
	}

	return nil
}

func evalPackages(ctx *flake.Context) ([]string, error) {
	out, err := exec.Command("nix", "eval", "--json",
		fmt.Sprintf("%s#nixosConfigurations.%s.config.environment.systemPackages", ctx.FlakePath, ctx.ConfigName),
		"--apply", "map (p: p.pname or p.name)",
	).Output()
	if err != nil {
		return nil, err
	}

	var names []string
	if err := json.Unmarshal(out, &names); err != nil {
		return nil, err
	}
	return names, nil
}

func match(rules []Rule, packages []string) []Match {
	var matches []Match
	for _, rule := range rules {
		for _, detectPkg := range rule.Detect.Packages {
			detect := strings.ToLower(detectPkg)
			for _, p := range packages {
				if strings.Contains(strings.ToLower(p), detect) {
					matches = append(matches, Match{Rule: rule, Suggestions: rule.Suggest})
					goto nextRule
				}
			}
		}
	nextRule:
	}
	return matches
}

func printMatches(matches []Match) {
	fmt.Println(color.New(color.Bold).Sprint("  Optimisations"))
	for _, m := range matches {
		fmt.Printf("  %s  %s\n", color.CyanString(""), m.Rule.Name)
		fmt.Printf("      %s\n", color.New(color.Faint).Sprint(m.Rule.Description))
		for k, v := range m.Suggestions {
			v = strings.TrimSpace(v)
			if strings.Contains(v, "\n") {
				lines := strings.Split(v, "\n")
				fmt.Printf("      %s %s = %s\n", color.New(color.Faint).Sprint("·"), color.YellowString(k), lines[0])
				for _, line := range lines[1:] {
					fmt.Printf("               %s\n", color.New(color.Faint).Sprint(line))
				}
			} else {
				fmt.Printf("      %s %s = %s\n", color.New(color.Faint).Sprint("·"), color.YellowString(k), color.New(color.Faint).Sprint(v))
			}
		}
		fmt.Println()
	}
}

func writeModules(matches []Match, flakePath string) error {
	noleDir := filepath.Join(flakePath, "nole")
	optDir := filepath.Join(noleDir, "optimizations")
	if err := os.MkdirAll(optDir, 0755); err != nil {
		return err
	}

	for _, m := range matches {
		path := filepath.Join(optDir, m.Rule.ID+".nix")
		if err := writeOptimizationModule(m, path); err != nil {
			return err
		}
	}

	if err := writeDefaultModule(matches, noleDir); err != nil {
		return err
	}

	if err := writeSuggestions(matches, noleDir); err != nil {
		return err
	}

	fmt.Println(color.New(color.Bold).Sprint("  Generated"))
	for _, m := range matches {
		fmt.Printf("  %s nole/optimizations/%s.nix\n", color.CyanString("󰈔"), m.Rule.ID)
	}
	fmt.Printf("  %s nole/default.nix\n", color.CyanString("󰈔"))
	fmt.Printf("  %s nole/README.md\n\n", color.CyanString("󰈔"))
	fmt.Printf("  %s Import %s in your flake, then enable via:\n",
		color.New(color.Faint).Sprint("→"),
		color.CyanString("./nole"),
	)
	for _, m := range matches {
		fmt.Printf("  %s modules.optimizations.%s.enable = true;\n",
			color.New(color.Faint).Sprint(" "),
			m.Rule.ID,
		)
	}
	fmt.Println()

	return nil
}

func writeOptimizationModule(m Match, path string) error {
	var sb strings.Builder
	sb.WriteString("{ lib, config, ... }:\n\n")
	sb.WriteString(fmt.Sprintf("lib.mkIf config.modules.optimizations.\"%s\".enable {\n", m.Rule.ID))

	for k, v := range m.Suggestions {
		v = strings.TrimSpace(v)
		if strings.Contains(v, "\n") {
			indented := "    " + strings.ReplaceAll(v, "\n", "\n    ")
			sb.WriteString(fmt.Sprintf("  %s = lib.mkDefault\n%s;\n", k, indented))
		} else {
			sb.WriteString(fmt.Sprintf("  %s = lib.mkDefault %s;\n", k, v))
		}
	}

	sb.WriteString("}\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeDefaultModule(matches []Match, noleDir string) error {
	var sb strings.Builder
	sb.WriteString("{ lib, ... }:\n\n")
	sb.WriteString("{\n")
	sb.WriteString("  imports = [\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("    ./optimizations/%s.nix\n", m.Rule.ID))
	}
	sb.WriteString("  ];\n\n")
	sb.WriteString("  options.modules.optimizations = {\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("    \"%s\".enable = lib.mkEnableOption \"%s optimisations\";\n", m.Rule.ID, m.Rule.Name))
	}
	sb.WriteString("  };\n")
	sb.WriteString("}\n")

	return os.WriteFile(filepath.Join(noleDir, "default.nix"), []byte(sb.String()), 0644)
}

func writeSuggestions(matches []Match, noleDir string) error {
	var sb strings.Builder
	sb.WriteString("# Nole Optimisations\n\n")
	sb.WriteString("Generated by `nole analyse`. Add `./nole` to your flake imports, then enable any detected optimisations in your host configuration.\n\n")
	sb.WriteString("## Detected\n\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("### %s\n\n", m.Rule.Name))
		sb.WriteString(fmt.Sprintf("%s\n\n", m.Rule.Description))
		sb.WriteString("```nix\n")
		sb.WriteString(fmt.Sprintf("modules.optimizations.\"%s\".enable = true;\n", m.Rule.ID))
		sb.WriteString("```\n\n")
		sb.WriteString("**Sets:**\n\n")
		for k, v := range m.Suggestions {
			sb.WriteString(fmt.Sprintf("- `%s = %s`\n", k, strings.TrimSpace(v)))
		}
		sb.WriteString("\n")
	}

	return os.WriteFile(filepath.Join(noleDir, "README.md"), []byte(sb.String()), 0644)
}
