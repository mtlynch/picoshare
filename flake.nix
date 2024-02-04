{
  description = "Dev environment for PicoShare";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";

    # 1.21.1 release
    go_dep.url = "github:NixOS/nixpkgs/78058d810644f5ed276804ce7ea9e82d92bee293";

    # 3.44.2 release
    sqlite_dep.url = "github:NixOS/nixpkgs/5ad9903c16126a7d949101687af0aa589b1d7d3d";

    # 20.6.1 release
    nodejs_dep.url = "github:NixOS/nixpkgs/78058d810644f5ed276804ce7ea9e82d92bee293";

    # 0.9.0 release
    shellcheck_dep.url = "github:NixOS/nixpkgs/8b5ab8341e33322e5b66fb46ce23d724050f6606";

    # 1.2.1 release
    sqlfluff_dep.url = "github:NixOS/nixpkgs/7cf5ccf1cdb2ba5f08f0ac29fc3d04b0b59a07e4";

    # 1.40.0
    playwright_dep.url = "github:NixOS/nixpkgs/f5c27c6136db4d76c30e533c20517df6864c46ee";

    # 0.1.131 release
    flyctl_dep.url = "github:NixOS/nixpkgs/09dc04054ba2ff1f861357d0e7e76d021b273cd7";

    # 0.3.9 release
    litestream_dep.url = "github:NixOS/nixpkgs/9d757ec498666cc1dcc6f2be26db4fd3e1e9ab37";
  };

  outputs = { self, flake-utils, go_dep, sqlite_dep, nodejs_dep, shellcheck_dep, sqlfluff_dep, playwright_dep, flyctl_dep, litestream_dep }@inputs :
    flake-utils.lib.eachDefaultSystem (system:
    let
      go_dep = inputs.go_dep.legacyPackages.${system};
      sqlite_dep = inputs.sqlite_dep.legacyPackages.${system};
      nodejs_dep = inputs.nodejs_dep.legacyPackages.${system};
      shellcheck_dep = inputs.shellcheck_dep.legacyPackages.${system};
      sqlfluff_dep = inputs.sqlfluff_dep.legacyPackages.${system};
      playwright_dep = inputs.playwright_dep.legacyPackages.${system};
      flyctl_dep = inputs.flyctl_dep.legacyPackages.${system};
      litestream_dep = inputs.litestream_dep.legacyPackages.${system};
    in
    {
      devShells.default = go_dep.mkShell.override { stdenv = go_dep.pkgsStatic.stdenv; } {
        packages = [
          go_dep.gotools
          go_dep.gopls
          go_dep.go-outline
          go_dep.gocode
          go_dep.gopkgs
          go_dep.gocode-gomod
          go_dep.godef
          go_dep.golint
          go_dep.go_1_21
          sqlite_dep.sqlite
          nodejs_dep.nodejs_20
          shellcheck_dep.shellcheck
          sqlfluff_dep.sqlfluff
          playwright_dep.playwright-driver.browsers
          flyctl_dep.flyctl
          litestream_dep.litestream
        ];

        shellHook = ''
          GOROOT="$(dirname $(dirname $(which go)))/share/go"
          export GOROOT

          export PLAYWRIGHT_BROWSERS_PATH=${playwright_dep.playwright-driver.browsers}
          export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true

          echo "shellcheck" "$(shellcheck --version | grep '^version:')"
          sqlfluff --version
          fly version | cut -d ' ' -f 1-3
          echo "litestream" "$(litestream version)"
          echo "node" "$(node --version)"
          echo "npm" "$(npm --version)"
          echo "sqlite" "$(sqlite3 --version | cut -d ' ' -f 1-2)"
          go version
        '';
      };
    });
}
