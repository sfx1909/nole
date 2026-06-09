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
      vendorHash = null;
    };

    devShells.${system}.default = pkgs.mkShell {
      buildInputs = [ pkgs.go pkgs.gopls pkgs.gotools ];
    };
  };
}
