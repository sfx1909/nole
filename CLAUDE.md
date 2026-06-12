# nole

A NixOS configuration manager, modeled as the NixOS counterpart to
[tw93/Mole](https://github.com/tw93/Mole) (`mo`), a macOS maintenance CLI.

## mole → nole feature map

| mole command/feature | nole equivalent | Status | Notes |
|---|---|---|---|
| `mo clean` (cache/log cleanup) | `nole clean` | Done | `nix-collect-garbage -d` + `nix store optimise`, preview by default, `--apply` to run |
| `mo optimize` | `nole clean` | Done | Folded into `clean` — store optimisation is part of the same maintenance pass |
| `mo uninstall` | — | Future | NixOS is declarative; "uninstall" would mean removing a package from config + rebuild. Could integrate with `nole analyse` |
| `mo analyze` (disk usage explorer) | `nole purge` (artifact sizes) / `nole analyse` (config audit) | Done | `nole analyse` audits package config for optimisations (different purpose); `nole purge` reports dev-artifact disk usage. A general interactive store explorer is Future |
| `mo status` (live dashboard) | `nole status` | Done | One-shot dashboard (generations, `/nix/store` usage, git/flake.lock state), not live/refreshing |
| `mo purge` (dev build artifacts) | `nole purge [path]` | Done | Finds `node_modules`, `target`, `dist`, `build`, `.direnv`, `__pycache__`, `.venv`, nix `result*` symlinks, etc. |
| `mo installer` | — | N/A | NixOS systems are installed via the NixOS installer/ISO + flake deploy; `nole` operates on an already-installed system |
| `mo touchid` | — | N/A | Touch ID is a macOS/Apple Silicon sudo-auth feature; `nole` already prompts for the sudo password directly (`builder.EnsureSudo`) |
| `mo completion` | `nole completion` | Done | Provided automatically by Cobra: `nole completion bash\|zsh\|fish\|powershell` |
| `mo update` | `nole maintain` | Done | Updates flake inputs and rebuilds only if the resulting system actually changes; `--clean` also runs garbage collection/store optimisation afterwards, and a tip is shown when garbage has piled up |
| `mo history` | `nole history` | Done | Reads the JSON-lines operations log written by `clean --apply` / `purge --apply` |
| global `--dry-run` | per-command `--apply` | Done (pattern) | Matches the existing `analyse --apply` convention: commands preview by default, `--apply`/`-a` executes. No global flag |
| global `--json` | — | Future | Would need structured output per command |
| global `--debug` | — | Future | Verbose logging of shelled-out commands |
| operations log (`~/Library/Logs/mole/operations.log`) | `~/.local/state/nole/operations.log` (or `$XDG_STATE_HOME/nole/operations.log`) | Done | JSON Lines via `internal/oplog`, surfaced via `nole history` |

## Conventions

- Cobra subcommands in `cmd/*.go` are thin wrappers that delegate to `internal/<pkg>.Run(...)`.
- Output uses `github.com/fatih/color` with 2-space-indented "  Bold header" blocks and
  Nerd Font icons, and `github.com/briandowns/spinner` for long-running operations.
- Destructive commands (`clean`, `purge`) preview by default and require `--apply`/`-a`
  to execute, prompting via `internal/git.Confirm` and recording an entry via
  `internal/oplog` on success.
- `flake.nix` is built with `pkgs.buildGoModule`, which pins `vendorHash`. Any change to
  `go.mod`/`go.sum` (e.g. adding/removing a dependency) requires updating `vendorHash` in
  `flake.nix`, otherwise `nix build`/`nole maintain` for this repo will fail with a hash
  mismatch.
