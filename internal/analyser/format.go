package analyser

import "fmt"

// Format selects the shape of the generated output under ./nole.
type Format string

const (
	// FormatModule generates a classic modules.optimizations.<id>.enable
	// tree (lib.mkIf-gated, with options.modules.optimizations in
	// nole/default.nix). This is the original/default format.
	FormatModule Format = "module"

	// FormatFlakePart generates flake.modules.nixos.<id> attrsets, one
	// per file, suitable for import-tree / Dendritic-style flake-parts
	// configs. Opt-in happens by listing the module in a host's module
	// list, not via an enable option.
	FormatFlakePart Format = "flake-part"

	// FormatFlake generates a small standalone flake under ./nole that
	// exposes nixosModules.<id> (+ nixosModules.default), so it can be
	// used as its own flake input.
	FormatFlake Format = "flake"
)

// ParseFormat validates a format string from a flag or config file.
// An empty string defaults to FormatModule.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case "":
		return FormatModule, nil
	case FormatModule, FormatFlakePart, FormatFlake:
		return Format(s), nil
	default:
		return "", fmt.Errorf("unknown format %q (expected module, flake-part, or flake)", s)
	}
}
