{
  description = "Dev environment for PicoShare";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";

    # 1.22.3 release
    go-nixpkgs.url = "github:NixOS/nixpkgs/a9858885e197f984d92d7fe64e9fff6b2e488d40";

    # 3.44.2 release
    sqlite-nixpkgs.url = "github:NixOS/nixpkgs/5ad9903c16126a7d949101687af0aa589b1d7d3d";
  };

  outputs = {
    self,
    flake-utils,
    go-nixpkgs,
    sqlite-nixpkgs,
  } @ inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      go = go-nixpkgs.legacyPackages.${system}.go;
      sqlite = sqlite-nixpkgs.legacyPackages.${system}.sqlite;
    in {
      devShells.default =
        go-nixpkgs.legacyPackages.${system}.mkShell.override
        {
          stdenv = go-nixpkgs.legacyPackages.${system}.pkgsStatic.stdenv;
        }
        {
          packages = [
            go-nixpkgs.legacyPackages.${system}.gotools
            go-nixpkgs.legacyPackages.${system}.gopls
            go-nixpkgs.legacyPackages.${system}.go-outline
            go-nixpkgs.legacyPackages.${system}.gopkgs
            go-nixpkgs.legacyPackages.${system}.gocode-gomod
            go-nixpkgs.legacyPackages.${system}.godef
            go-nixpkgs.legacyPackages.${system}.golint
            go
            sqlite
          ];

          shellHook = ''
            export GOROOT="${go}/share/go"

            echo "sqlite" "$(sqlite3 --version | cut -d ' ' -f 1-2)"
            go version
          '';
        };

      formatter = go-nixpkgs.legacyPackages.${system}.alejandra;
    });
}
