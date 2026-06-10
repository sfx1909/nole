{
  description = "Nole - NixOS configuration manager and optimiser";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
  let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
    version = "0.2.0";
  in
  {
    packages.${system}.default = pkgs.buildGoModule {
      pname = "nole";
      inherit version;
      src = ./.;
      vendorHash = "sha256-e51nbkepzKB4mXwq+bXGHIElQgGoEqJ63x7d6tbLpFE=";
      ldflags = [ "-X github.com/sfx1909/nole/cmd.version=${version}" ];
    };

    nixosModules.default = { lib, config, pkgs, ... }:
    let
      cfg = config.programs.nole;
    in {
      options.programs.nole = {
        enable = lib.mkEnableOption "nole NixOS manager";

        flakePath = lib.mkOption {
          type = lib.types.str;
          description = ''
            Path to your NixOS flake, optionally suffixed with
            "#<configuration>" to pin the nixosConfigurations attribute
            (e.g. "/home/you/nixos-config#hostname"). If omitted, nole
            resolves the configuration automatically.
          '';
        };

        format = lib.mkOption {
          type = lib.types.enum [ "module" "flake-part" "flake" ];
          default = "module";
          description = "Default output format for `nole analyse --apply`.";
        };

        maintain.clean = lib.mkOption {
          type = lib.types.bool;
          default = false;
          description = ''
            Whether `nole maintain` should also garbage-collect old
            generations and optimise the Nix store by default
            (equivalent to always passing --clean).
          '';
        };
      };

      config = lib.mkIf cfg.enable {
        environment.systemPackages = [ self.packages.${pkgs.system}.default ];
        environment.etc."nole/config.toml".source = (pkgs.formats.toml { }).generate "nole-config.toml" {
          flake = cfg.flakePath;
          format = cfg.format;
          maintain.clean = cfg.maintain.clean;
        };
      };
    };

    devShells.${system}.default = pkgs.mkShell {
      buildInputs = [ pkgs.go pkgs.gopls pkgs.gotools ];
    };
  };
}
