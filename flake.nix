{
  description = "Nole - NixOS configuration manager and optimiser";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
  let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in
  {
    packages.${system}.default = pkgs.buildGoModule {
      pname = "nole";
      version = "0.1.0";
      src = ./.;
      vendorHash = "";
    };

    nixosModules.default = { lib, config, pkgs, ... }:
    let
      cfg = config.programs.nole;
    in {
      options.programs.nole = {
        enable = lib.mkEnableOption "nole NixOS manager";
        flakePath = lib.mkOption {
          type = lib.types.str;
          description = "Absolute path to your NixOS flake";
        };
      };

      config = lib.mkIf cfg.enable {
        environment.systemPackages = [ self.packages.${pkgs.system}.default ];
        environment.etc."nole/config.toml".text = ''
          flake = "${cfg.flakePath}"
        '';
      };
    };

    devShells.${system}.default = pkgs.mkShell {
      buildInputs = [ pkgs.go pkgs.gopls pkgs.gotools ];
    };
  };
}
