{
  description = "Dev environment for PicoShare";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";

    # 1.24.0 release
    go-nixpkgs.url = "github:NixOS/nixpkgs/83a2581c81ff5b06f7c1a4e7cc736a455dfcf7b4";

    # 3.44.2 release
    sqlite-nixpkgs.url = "github:NixOS/nixpkgs/5ad9903c16126a7d949101687af0aa589b1d7d3d";

    # 20.6.1 release
    nodejs-nixpkgs.url = "github:NixOS/nixpkgs/78058d810644f5ed276804ce7ea9e82d92bee293";

    # 0.9.0 release
    shellcheck-nixpkgs.url = "github:NixOS/nixpkgs/8b5ab8341e33322e5b66fb46ce23d724050f6606";

    # 3.1.1 release
    sqlfluff-nixpkgs.url = "github:NixOS/nixpkgs/5629520edecb69630a3f4d17d3d33fc96c13f6fe";

    # 0.1.131 release
    flyctl-nixpkgs.url = "github:NixOS/nixpkgs/09dc04054ba2ff1f861357d0e7e76d021b273cd7";

    # 0.3.13 release
    litestream-nixpkgs.url = "github:NixOS/nixpkgs/a343533bccc62400e8a9560423486a3b6c11a23b";
  };

  outputs = {
    self,
    flake-utils,
    go-nixpkgs,
    sqlite-nixpkgs,
    nodejs-nixpkgs,
    shellcheck-nixpkgs,
    sqlfluff-nixpkgs,
    flyctl-nixpkgs,
    litestream-nixpkgs,
  } @ inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      go-pkgs = go-nixpkgs.legacyPackages.${system};
      go = go-pkgs.go_1_24;
      sqlite = sqlite-nixpkgs.legacyPackages.${system}.sqlite;
      nodejs-pkgs = nodejs-nixpkgs.legacyPackages.${system};
      nodejs = nodejs-pkgs.nodejs_20;
      shellcheck = shellcheck-nixpkgs.legacyPackages.${system}.shellcheck;
      sqlfluff = sqlfluff-nixpkgs.legacyPackages.${system}.sqlfluff;
      flyctl = flyctl-nixpkgs.legacyPackages.${system}.flyctl;
      litestream = litestream-nixpkgs.legacyPackages.${system}.litestream;
      picoshare-dev = go-pkgs.buildGo124Module {
        pname = "picoshare-dev";
        version = "0.0.0";
        src = go-pkgs.lib.cleanSource ./.;
        subPackages = [ "cmd/picoshare" ];
        tags = [ "netgo" "sqlite_omit_load_extension" "dev" ];
        preBuild = ''
          ulimit -u 4096 || true
        '';
        env = {
          CGO_ENABLED = "1";
          GOMAXPROCS = "1";
          NIX_BUILD_CORES = "1";
        };
        GOFLAGS = [ "-mod=vendor" "-trimpath" "-p=1" ];
        buildInputs = [ sqlite ];
        ldflags = [
          "-w"
          "-X=github.com/mtlynch/picoshare/build.Version=dev"
          "-X=github.com/mtlynch/picoshare/build.unixTime=0"
        ];
        vendorHash = "sha256-X2vrEhgEnKKNXRyLCtT+wBbunFHgkcyWZh6DMpQieQ0=";
      };
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
            nodejs
            shellcheck
            sqlfluff
            flyctl
            litestream
          ];

          shellHook = ''
            # Avoid sharing GOPATH with other projects.
            PROJECT_NAME="$(basename "$PWD")"
            export GOPATH="$HOME/.local/share/go-workspaces/$PROJECT_NAME"

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

      formatter = go-nixpkgs.legacyPackages.${system}.alejandra;

      packages = {
        picoshare-dev = picoshare-dev;
      };
    });
}
