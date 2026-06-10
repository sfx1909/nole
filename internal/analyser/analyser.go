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
	"github.com/charmbracelet/huh"
	"github.com/sfx1909/nole/internal/flake"
	"github.com/sfx1909/nole/internal/style"
)

type Match struct {
	Rule        Rule
	Suggestions map[string]string
}

func Run(apply bool, format Format) error {
	ctx, err := flake.Detect()
	if err != nil {
		return err
	}

	rules, err := loadRules()
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Suffix = style.Faint.Render("  Evaluating config")
	s.Start()

	packages, err := evalPackages(ctx)
	s.Stop()
	if err != nil {
		return fmt.Errorf("failed to evaluate packages: %w", err)
	}

	matches := match(rules, packages)

	if len(matches) == 0 {
		fmt.Println(style.Green.Render("  󰄬  No optimisations found"))
		fmt.Println()
		return nil
	}

	printMatches(matches)

	if apply {
		if err := writeModules(matches, ctx.FlakePath, format); err != nil {
			return fmt.Errorf("failed to write optimisation modules: %w", err)
		}
		return nil
	}

	proceed, selected, err := selectMatches(matches)
	if err != nil {
		return fmt.Errorf("failed to read selection: %w", err)
	}

	if !proceed {
		fmt.Printf("  %s Run with %s to apply all without prompting\n\n",
			style.Faint.Render("→"),
			style.Cyan.Render("--apply"),
		)
		return nil
	}

	if len(selected) == 0 {
		fmt.Println(style.Faint.Render("  Nothing selected"))
		fmt.Println()
		return nil
	}

	if err := writeModules(selected, ctx.FlakePath, format); err != nil {
		return fmt.Errorf("failed to write optimisation modules: %w", err)
	}

	return nil
}

// selectMatches prompts the user, via a single themed huh form, whether to
// generate NixOS modules and (if so) which detected optimisations to include
// via a checkbox list (all selected by default).
func selectMatches(matches []Match) (proceed bool, selected []Match, err error) {
	proceed = true

	options := make([]huh.Option[int], len(matches))
	picked := make([]int, len(matches))
	for i, m := range matches {
		options[i] = huh.NewOption(fmt.Sprintf("%s — %s", m.Rule.Name, m.Rule.Description), i)
		picked[i] = i
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Generate NixOS modules for these optimisations?").
				Affirmative("Yes").
				Negative("No").
				Value(&proceed),
		),
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select optimisations to generate").
				Options(options...).
				Value(&picked),
		).WithHideFunc(func() bool { return !proceed }),
	)

	if err := style.RunForm(form); err != nil {
		return false, nil, err
	}

	if !proceed {
		return false, nil, nil
	}

	selected = make([]Match, 0, len(picked))
	for _, i := range picked {
		selected = append(selected, matches[i])
	}
	return true, selected, nil
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
	fmt.Println(style.Bold.Render("  Optimisations"))
	for _, m := range matches {
		fmt.Printf("  %s  %s\n", style.Cyan.Render(""), m.Rule.Name)
		fmt.Printf("      %s\n", style.Faint.Render(m.Rule.Description))
		for k, v := range m.Suggestions {
			v = strings.TrimSpace(v)
			if strings.Contains(v, "\n") {
				lines := strings.Split(v, "\n")
				fmt.Printf("      %s %s = %s\n", style.Faint.Render("·"), style.Yellow.Render(k), lines[0])
				for _, line := range lines[1:] {
					fmt.Printf("               %s\n", style.Faint.Render(line))
				}
			} else {
				fmt.Printf("      %s %s = %s\n", style.Faint.Render("·"), style.Yellow.Render(k), style.Faint.Render(v))
			}
		}
		fmt.Println()
	}
}

func writeModules(matches []Match, flakePath string, format Format) error {
	noleDir := filepath.Join(flakePath, "nole")
	if err := os.MkdirAll(noleDir, 0755); err != nil {
		return err
	}

	switch format {
	case FormatFlakePart:
		if err := writeFlakePartModules(matches, noleDir); err != nil {
			return err
		}
	case FormatFlake:
		if err := writeStandaloneFlake(matches, noleDir); err != nil {
			return err
		}
	default:
		if err := writeOptimizationModules(matches, noleDir); err != nil {
			return err
		}
	}

	printGenerated(matches, format)
	return nil
}

// writeAttrs writes `key = lib.mkDefault value;` lines for each
// suggestion, indented by indent. Multi-line values are reindented to
// indent+"  ".
func writeAttrs(sb *strings.Builder, suggestions map[string]string, indent string) {
	for k, v := range suggestions {
		v = strings.TrimSpace(v)
		if strings.Contains(v, "\n") {
			inner := indent + "  "
			indented := inner + strings.ReplaceAll(v, "\n", "\n"+inner)
			sb.WriteString(fmt.Sprintf("%s%s = lib.mkDefault\n%s;\n", indent, k, indented))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s = lib.mkDefault %s;\n", indent, k, v))
		}
	}
}

// writeDetectedSection appends the shared "## Detected" README section
// describing each match and the config it sets. If snippet is non-nil,
// its result is inserted (as a fenced nix block) before "**Sets:**" for
// each match.
func writeDetectedSection(sb *strings.Builder, matches []Match, snippet func(m Match) string) {
	sb.WriteString("## Detected\n\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("### %s\n\n", m.Rule.Name))
		sb.WriteString(fmt.Sprintf("%s\n\n", m.Rule.Description))
		if snippet != nil {
			sb.WriteString(fmt.Sprintf("```nix\n%s\n```\n\n", snippet(m)))
		}
		sb.WriteString("**Sets:**\n\n")
		for k, v := range m.Suggestions {
			sb.WriteString(fmt.Sprintf("- `%s = %s`\n", k, strings.TrimSpace(v)))
		}
		sb.WriteString("\n")
	}
}

func printGenerated(matches []Match, format Format) {
	fmt.Println(style.Bold.Render("  Generated"))

	switch format {
	case FormatFlakePart:
		for _, m := range matches {
			fmt.Printf("  %s nole/%s.nix\n", style.Cyan.Render("󰈔"), m.Rule.ID)
		}
		fmt.Printf("  %s nole/README.md\n\n", style.Cyan.Render("󰈔"))
		fmt.Printf("  %s Add %s to your flake-parts imports (e.g. via import-tree),\n", style.Faint.Render("→"), style.Cyan.Render("./nole"))
		fmt.Printf("  %s then reference these in a host's module list:\n", style.Faint.Render(" "))
		for _, m := range matches {
			fmt.Printf("  %s config.flake.nixosModules.\"%s\"\n", style.Faint.Render(" "), m.Rule.ID)
		}
	case FormatFlake:
		fmt.Printf("  %s nole/flake.nix\n", style.Cyan.Render("󰈔"))
		fmt.Printf("  %s nole/README.md\n\n", style.Cyan.Render("󰈔"))
		fmt.Printf("  %s Add as a flake input, e.g.:\n", style.Faint.Render("→"))
		fmt.Printf("  %s nole-optimizations.url = \"path:./nole\";\n", style.Faint.Render(" "))
		fmt.Printf("  %s then import inputs.nole-optimizations.nixosModules.default\n", style.Faint.Render(" "))
		fmt.Printf("  %s (or pick individual modules by id)\n", style.Faint.Render(" "))
	default:
		for _, m := range matches {
			fmt.Printf("  %s nole/optimizations/%s.nix\n", style.Cyan.Render("󰈔"), m.Rule.ID)
		}
		fmt.Printf("  %s nole/default.nix\n", style.Cyan.Render("󰈔"))
		fmt.Printf("  %s nole/README.md\n\n", style.Cyan.Render("󰈔"))
		fmt.Printf("  %s Import %s in your flake, then enable via:\n", style.Faint.Render("→"), style.Cyan.Render("./nole"))
		for _, m := range matches {
			fmt.Printf("  %s modules.optimizations.%s.enable = true;\n", style.Faint.Render(" "), m.Rule.ID)
		}
	}
	fmt.Println()
}

// --- module format (default): modules.optimizations.<id>.enable, lib.mkIf-gated ---

func writeOptimizationModules(matches []Match, noleDir string) error {
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

	return writeReadmeModule(matches, noleDir)
}

func writeOptimizationModule(m Match, path string) error {
	var sb strings.Builder
	sb.WriteString("{ lib, config, ... }:\n\n")
	sb.WriteString(fmt.Sprintf("lib.mkIf config.modules.optimizations.\"%s\".enable {\n", m.Rule.ID))
	writeAttrs(&sb, m.Suggestions, "  ")
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

func writeReadmeModule(matches []Match, noleDir string) error {
	var sb strings.Builder
	sb.WriteString("# Nole Optimisations\n\n")
	sb.WriteString("Generated by `nole analyse`. Add `./nole` to your flake imports, then enable any detected optimisations in your host configuration.\n\n")
	writeDetectedSection(&sb, matches, func(m Match) string {
		return fmt.Sprintf("modules.optimizations.\"%s\".enable = true;", m.Rule.ID)
	})

	return os.WriteFile(filepath.Join(noleDir, "README.md"), []byte(sb.String()), 0644)
}

// --- flake-part format: flake.nixosModules.<id>, for import-tree / flake-parts configs ---

func writeFlakePartModules(matches []Match, noleDir string) error {
	for _, m := range matches {
		path := filepath.Join(noleDir, m.Rule.ID+".nix")
		if err := writeFlakePartModule(m, path); err != nil {
			return err
		}
	}

	return writeReadmeFlakePart(matches, noleDir)
}

func writeFlakePartModule(m Match, path string) error {
	var sb strings.Builder
	sb.WriteString("{ lib, ... }:\n\n")
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("  flake.nixosModules.\"%s\" = {\n", m.Rule.ID))
	writeAttrs(&sb, m.Suggestions, "    ")
	sb.WriteString("  };\n")
	sb.WriteString("}\n")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func writeReadmeFlakePart(matches []Match, noleDir string) error {
	var sb strings.Builder
	sb.WriteString("# Nole Optimisations (flake-parts)\n\n")
	sb.WriteString("Generated by `nole analyse --format=flake-part`. Each file in this\n")
	sb.WriteString("directory declares a `flake.nixosModules.\"<id>\"` attrset for use with\n")
	sb.WriteString("flake-parts (e.g. via `import-tree ./nole`).\n\n")
	sb.WriteString("Opt in per host by adding the module to that host's module list:\n\n")
	sb.WriteString("```nix\n")
	sb.WriteString("modules = [\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("  config.flake.nixosModules.\"%s\"\n", m.Rule.ID))
	}
	sb.WriteString("];\n")
	sb.WriteString("```\n\n")
	writeDetectedSection(&sb, matches, nil)

	return os.WriteFile(filepath.Join(noleDir, "README.md"), []byte(sb.String()), 0644)
}

// --- flake format: standalone flake exposing nixosModules.<id> + nixosModules.default ---

func writeStandaloneFlake(matches []Match, noleDir string) error {
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString("  description = \"Nole-generated NixOS optimisation modules\";\n\n")
	sb.WriteString("  outputs = { self, ... }: {\n")
	sb.WriteString("    nixosModules = {\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("      \"%s\" = { lib, ... }: {\n", m.Rule.ID))
		writeAttrs(&sb, m.Suggestions, "        ")
		sb.WriteString("      };\n")
	}
	sb.WriteString("\n      default = { imports = [\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("        self.nixosModules.\"%s\"\n", m.Rule.ID))
	}
	sb.WriteString("      ]; };\n")
	sb.WriteString("    };\n")
	sb.WriteString("  };\n")
	sb.WriteString("}\n")

	if err := os.WriteFile(filepath.Join(noleDir, "flake.nix"), []byte(sb.String()), 0644); err != nil {
		return err
	}

	return writeReadmeFlake(matches, noleDir)
}

func writeReadmeFlake(matches []Match, noleDir string) error {
	var sb strings.Builder
	sb.WriteString("# Nole Optimisations (standalone flake)\n\n")
	sb.WriteString("Generated by `nole analyse --format=flake`. `./nole/flake.nix` exposes\n")
	sb.WriteString("`nixosModules.\"<id>\"` for each detected optimisation, plus a combined\n")
	sb.WriteString("`nixosModules.default`.\n\n")
	sb.WriteString("Use it as a flake input:\n\n")
	sb.WriteString("```nix\n")
	sb.WriteString("inputs.nole-optimizations.url = \"path:./nole\";\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Then import it in a host's module list, either the combined default:\n\n")
	sb.WriteString("```nix\n")
	sb.WriteString("inputs.nole-optimizations.nixosModules.default\n")
	sb.WriteString("```\n\n")
	sb.WriteString("or individual modules:\n\n")
	sb.WriteString("```nix\n")
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("inputs.nole-optimizations.nixosModules.\"%s\"\n", m.Rule.ID))
	}
	sb.WriteString("```\n\n")
	writeDetectedSection(&sb, matches, nil)

	return os.WriteFile(filepath.Join(noleDir, "README.md"), []byte(sb.String()), 0644)
}
