# nole

A NixOS configuration manager. Handles rebuilds, flake maintenance, and config optimisation detection.

## Commands

### `nole build`
Rebuilds your NixOS configuration.

- Checks for untracked `.nix` files before building (untracked files are invisible to the nix evaluator)
- Shows a spinner while building with a summary of warnings and packages built
- After a successful build, prompts to stage and commit any changed `.nix` files

### `nole maintain`
Updates flake inputs and rebuilds only if needed.

- Updates all flake inputs
- Auto-commits `flake.lock` if it's the only changed file
- Compares derivation paths to detect if a rebuild is actually needed
- Skips the rebuild if the system is already up to date
- Prompts to stage and commit any changed `.nix` files

### `nole analyse`
Detects installed packages and suggests NixOS optimisations.

- Evaluates your system packages
- Matches against known optimisation rules (OBS, gaming, COSMIC, PipeWire, Docker)
- Run with `--apply` to generate ready-to-import NixOS modules under `./nole/optimizations/`

## Installation

Add nole to your NixOS flake:

```nix
# flake.nix
inputs = {
  nole.url = "github:sfx1909/nole";
  nole.inputs.nixpkgs.follows = "nixpkgs";
};
```

Import the module and enable it in your host configuration:

```nix
# configuration.nix or equivalent
imports = [ inputs.nole.nixosModules.default ];

programs.nole = {
  enable = true;
  flakePath = "/home/you/nixos-config";
};
```

This installs the `nole` binary and writes `/etc/nole/config.toml` automatically.

## Configuration

Config is loaded in priority order:

1. `$XDG_CONFIG_HOME/nole/config.toml` (or `~/.config/nole/config.toml`)
2. `/etc/nole/config.toml` (written by the NixOS module)

```toml
flake = "/home/you/nixos-config"
```

The flake path can also be set per-invocation via the `NOLE_FLAKE` environment variable.
