{
  description = "Dev environment for PicoShare";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";

    # Use https://www.nixhub.io/ to find the right nixpkgs commit for the
    # specific package version we want.

    # 1.26.0 release
    go-nixpkgs.url = "github:NixOS/nixpkgs/a82ccc39b39b621151d6732718e3e250109076fa";

    # 3.44.2 release
    sqlite-nixpkgs.url = "github:NixOS/nixpkgs/5ad9903c16126a7d949101687af0aa589b1d7d3d";

    # 20.6.1 release
    nodejs-nixpkgs.url = "github:NixOS/nixpkgs/78058d810644f5ed276804ce7ea9e82d92bee293";

    # 0.9.0 release
    shellcheck-nixpkgs.url = "github:NixOS/nixpkgs/8b5ab8341e33322e5b66fb46ce23d724050f6606";

    # 3.1.1 release
    sqlfluff-nixpkgs.url = "github:NixOS/nixpkgs/5629520edecb69630a3f4d17d3d33fc96c13f6fe";

    # 1.59.1
    playwright-nixpkgs.url = "github:NixOS/nixpkgs/7aaa00e7cc9be6c316cb5f6617bd740dd435c59d";

    # 0.1.131 release
    flyctl-nixpkgs.url = "github:NixOS/nixpkgs/09dc04054ba2ff1f861357d0e7e76d021b273cd7";

    # 0.3.13 release
    litestream-nixpkgs.url = "github:NixOS/nixpkgs/a343533bccc62400e8a9560423486a3b6c11a23b";

    # 1.61.7 release
    air-nixpkgs.url = "github:NixOS/nixpkgs/678af34d5e4d198fcd948a7db8b89a618d8e62fa";
  };

  outputs = {
    self,
    flake-utils,
    go-nixpkgs,
    sqlite-nixpkgs,
    nodejs-nixpkgs,
    shellcheck-nixpkgs,
    sqlfluff-nixpkgs,
    playwright-nixpkgs,
    flyctl-nixpkgs,
    litestream-nixpkgs,
    air-nixpkgs,
  } @ inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      gopkg = go-nixpkgs.legacyPackages.${system};
      go = gopkg.go_1_26;
      buildGoModule = gopkg.buildGoModule.override {
        inherit go;
        stdenv = gopkg.pkgsStatic.stdenv;
      };
      sqlite = sqlite-nixpkgs.legacyPackages.${system}.sqlite;
      nodepkgs = nodejs-nixpkgs.legacyPackages.${system};
      nodejs = nodepkgs.nodejs_20;
      shellcheck = shellcheck-nixpkgs.legacyPackages.${system}.shellcheck;
      sqlfluff = sqlfluff-nixpkgs.legacyPackages.${system}.sqlfluff;
      playwright = playwright-nixpkgs.legacyPackages.${system}.playwright-driver.browsers;
      flyctl = flyctl-nixpkgs.legacyPackages.${system}.flyctl;
      litestream = litestream-nixpkgs.legacyPackages.${system}.litestream;
      air = air-nixpkgs.legacyPackages.${system}.air;

      # Fonts for Playwright browser tests.
      fontsConf = nodepkgs.makeFontsConf {
        fontDirectories = [nodepkgs.dejavu_fonts];
      };

      goVendorHash = "sha256-X2vrEhgEnKKNXRyLCtT+wBbunFHgkcyWZh6DMpQieQ0=";

      npmDepsHash = "sha256-7z4Fdtl0WqriTyh9g1sUlNyoc/vyp5akeP0b/JDzheQ=";

      backend-dev = buildGoModule {
        pname = "picoshare-dev";
        version = "0.0.1";
        src = gopkg.lib.cleanSource ./.;
        vendorHash = goVendorHash;
        subPackages = ["cmd/picoshare"];
        env.CGO_ENABLED = "1";
        tags = ["netgo" "sqlite_omit_load_extension" "dev"];
        ldflags = ["-w" "-extldflags '-static'"];
        postInstall = ''
          mv "$out/bin/picoshare" "$out/bin/picoshare-dev"
        '';
      };
    in {
      packages = {
        inherit backend-dev;

        e2e-tests = nodepkgs.buildNpmPackage {
          pname = "picoshare-e2e-tests";
          version = "0.0.1";
          src = nodepkgs.lib.cleanSource ./.;
          inherit npmDepsHash;
          npmInstallFlags = ["--ignore-scripts"];
          dontNpmBuild = true;
          nativeBuildInputs = [nodejs playwright backend-dev];
          doCheck = true;
          checkPhase = ''
            export HOME="$PWD/.home"
            mkdir -p "$HOME"
            export CI=1
            export PLAYWRIGHT_BROWSERS_PATH=${playwright}
            export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
            export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true

            # Configure fonts for headless browser rendering.
            export FONTCONFIG_FILE=${fontsConf}

            mkdir -p ./bin
            cp ${backend-dev}/bin/picoshare-dev ./bin/picoshare-dev

            cd e2e
            npx playwright test --project=chromium
          '';
          installPhase = ''
            mkdir -p "$out"
            printf 'e2e-tests passed\n' > "$out/result"
          '';
        };
      };

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
            playwright
            flyctl
            litestream
            air
          ];

          shellHook = ''
            # Avoid sharing GOPATH with other projects.
            PROJECT_NAME="$(basename "$PWD")"
            export GOPATH="$HOME/.local/share/go-workspaces/$PROJECT_NAME"

            export PLAYWRIGHT_BROWSERS_PATH=${playwright}
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

      formatter = go-nixpkgs.legacyPackages.${system}.alejandra;
    });
}
