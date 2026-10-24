{
  description = "Dev environment for PicoShare";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";

    # Use https://www.nixhub.io/ to find the right nixpkgs commit for the
    # specific package version we want.

    # 1.26.1 release
    go-nixpkgs.url = "github:NixOS/nixpkgs/e607cb5360ff1234862ac9f8839522becb853bb9";

    # 3.51.1 release
    sqlite-nixpkgs.url = "github:NixOS/nixpkgs/13868c071cc73a5e9f610c47d7bb08e5da64fdd5";

    # 24.15.0 release
    nodejs-nixpkgs.url = "github:NixOS/nixpkgs/9eac87a12312b8f60dd52e1c6e1a265f6fc7f5fc";

    # 0.11.0 release
    shellcheck-nixpkgs.url = "github:NixOS/nixpkgs/1d4c88323ac36805d09657d13a5273aea1b34f0c";

    # 0.14.0 release
    gci-nixpkgs.url = "github:NixOS/nixpkgs/29916453413845e54a65b8a1cf996842300cd299";

    # 3.5.0 release
    sqlfluff-nixpkgs.url = "github:NixOS/nixpkgs/8b6600824693a9c706ef09bd86711ca393703466";

    # 1.59.1
    playwright-nixpkgs.url = "github:NixOS/nixpkgs/7aaa00e7cc9be6c316cb5f6617bd740dd435c59d";

    # 0.4.59 release
    flyctl-nixpkgs.url = "github:NixOS/nixpkgs/5a722a7155bfc9fbe657f28d26b71860d95324bc";

    # 0.5.11 release
    litestream-nixpkgs.url = "github:NixOS/nixpkgs/49a4bd0573c376468dd7996ddb6f9fa31d8c4d97";

    # 1.67.4 release
    air-nixpkgs.url = "github:NixOS/nixpkgs/f2154edcc6bedad158da01133795b2cfafb3fa6a";
  };

  outputs = {
    self,
    flake-utils,
    go-nixpkgs,
    sqlite-nixpkgs,
    nodejs-nixpkgs,
    shellcheck-nixpkgs,
    gci-nixpkgs,
    sqlfluff-nixpkgs,
    playwright-nixpkgs,
    flyctl-nixpkgs,
    litestream-nixpkgs,
    air-nixpkgs,
  } @ inputs:
    flake-utils.lib.eachDefaultSystem (system: let
      gopkg = import go-nixpkgs {
        inherit system;
        overlays = [
          (final: prev: {
            buildGoModule = prev.buildGoModule.override {go = final.go_1_26;};
          })
        ];
      };
      go = gopkg.go_1_26;
      buildGoModule = gopkg.buildGoModule.override {
        inherit go;
        stdenv = gopkg.pkgsStatic.stdenv;
      };
      sqlite = sqlite-nixpkgs.legacyPackages.${system}.sqlite;
      nodepkgs = nodejs-nixpkgs.legacyPackages.${system};
      nodejs = nodepkgs.nodejs_24;
      buildNpmPackage = nodepkgs.buildNpmPackage.override {inherit nodejs;};
      shellcheck = shellcheck-nixpkgs.legacyPackages.${system}.shellcheck;
      gci = gci-nixpkgs.legacyPackages.${system}.gci;
      sqlfluff = sqlfluff-nixpkgs.legacyPackages.${system}.sqlfluff;
      flyctl = flyctl-nixpkgs.legacyPackages.${system}.flyctl;
      litestream = litestream-nixpkgs.legacyPackages.${system}.litestream;
      air = air-nixpkgs.legacyPackages.${system}.air;

      # Go static analysis tools, including Staticcheck
      go-tools = gopkg.go-tools.override {buildGoModule = gopkg.buildGoModule;};
      errcheck = gopkg.errcheck.override {buildGoModule = gopkg.buildGoModule;};
      go-critic = gopkg.go-critic.override {buildGoModule = gopkg.buildGoModule;};

      goVendorHash = "sha256-QpeLePk1OcgNAnIPtOV5HSPq1YS/YX4TfIqJIio6Czc=";

      npmDepsHash = "sha256-Y/RE2xR1bBApeSqKG+6f89lQ8b6NAEE3lV0b0z2ACu4=";

      appName = "picoshare";
      appNameDev = "${appName}-dev";

      backendSrc = gopkg.lib.fileset.toSource {
        root = ./.;
        fileset = gopkg.lib.fileset.unions [
          ./go.mod
          ./go.sum
          (gopkg.lib.fileset.fileFilter (file: file.hasExt "go") ./.)
          ./handlers/static
          ./handlers/templates
          ./store/sqlite/migrations
        ];
      };

      npmDependenciesSrc = gopkg.lib.fileset.toSource {
        root = ./.;
        fileset = gopkg.lib.fileset.unions [
          ./package.json
          ./package-lock.json
        ];
      };

      npmDeps = nodepkgs.fetchNpmDeps {
        name = "picoshare-npm-deps";
        src = npmDependenciesSrc;
        hash = npmDepsHash;
      };

      mkBuildStep = {
        name,
        command,
        src ? gopkg.lib.cleanSource ./.,
        extraInputs ? [],
        setup ? "",
        extraAttrs ? {},
      }:
        gopkg.stdenvNoCC.mkDerivation ({
            pname = name;
            version = "0.0.1";
            inherit src;
            nativeBuildInputs = [gopkg.bash] ++ extraInputs;
            dontConfigure = true;
            buildPhase = ''
              runHook preBuild
              patchShebangs ./dev-scripts
              ${setup}
              ${command}
              runHook postBuild
            '';
            installPhase = ''
              touch "$out"
            '';
          }
          // extraAttrs);

      mkAppPackage = {
        pname,
        extraTags ? [],
      }:
        buildGoModule {
          inherit pname;
          version = "0.0.1";
          src = backendSrc;
          vendorHash = goVendorHash;
          subPackages = ["cmd/picoshare"];
          env.CGO_ENABLED = "1";
          tags = ["netgo" "sqlite_omit_load_extension"] ++ extraTags;
          ldflags = ["-w" "-extldflags '-static'"];
          postInstall = gopkg.lib.optionalString (pname != "picoshare") ''
            mv "$out/bin/picoshare" "$out/bin/${pname}"
          '';
        };

      appPackage = mkAppPackage {
        pname = appName;
      };

      appPackageDev = mkAppPackage {
        pname = appNameDev;
        extraTags = ["dev"];
      };

      frontend-check = buildNpmPackage {
        pname = "picoshare-frontend-check";
        version = "0.0.1";
        src = gopkg.lib.fileset.toSource {
          root = ./.;
          fileset =
            gopkg.lib.fileset.intersection
            (gopkg.lib.fileset.gitTracked ./.)
            (gopkg.lib.fileset.unions [
              ./dev-scripts/build-frontend
              ./.gitignore
              ./.prettierignore
              ./.prettierrc
              ./eslint.config.js
              ./package.json
              ./package-lock.json
              (gopkg.lib.fileset.fileFilter (file:
                file.hasExt "md"
                || file.hasExt "js"
                || file.hasExt "html"
                || file.hasExt "css"
                || file.hasExt "json"
                || file.hasExt "yaml"
                || file.hasExt "yml"
                || file.hasExt "ts")
              ./.)
            ]);
        };
        inherit npmDeps;
        npmInstallFlags = ["--ignore-scripts"];
        dontNpmBuild = true;
        nativeBuildInputs = [nodepkgs.git];
        doCheck = true;
        checkPhase = ''
          runHook preCheck
          git init --quiet
          git add --all
          patchShebangs ./dev-scripts
          ./dev-scripts/build-frontend
          runHook postCheck
        '';
        installPhase = ''
          mkdir -p "$out"
          printf 'frontend-check passed\n' > "$out/result"
        '';
      };
    in {
      packages = {
        "${appName}" = appPackage;
        "${appNameDev}" = appPackageDev;
        inherit frontend-check;

        go-tests = mkBuildStep {
          name = "go-tests";
          command = "./dev-scripts/run-go-tests";
          src = gopkg.lib.fileset.toSource {
            root = ./.;
            fileset = gopkg.lib.fileset.unions [
              ./go.mod
              ./go.sum
              ./dev-scripts/run-go-tests
              (gopkg.lib.fileset.fileFilter (file: file.hasExt "go") ./.)
              ./handlers/static
              ./handlers/templates
              ./store/sqlite/migrations
            ];
          };
          extraInputs = [
            go
            sqlite
            gopkg.git
            gopkg.gcc
            gopkg.binutils
            go-tools
            errcheck
            go-critic
            gci
          ];
          setup = ''
            export HOME="$TMPDIR/home"
            mkdir -p "$HOME"

            git init --quiet
            git add --all

            # Use pre-fetched Go modules to avoid network access.
            cp -r ${appPackage.goModules} vendor
            chmod -R u+w vendor
            export GOFLAGS="-mod=vendor"
          '';
        };

        check-bash = mkBuildStep {
          name = "check-bash";
          command = "./dev-scripts/check-bash";
          extraInputs = [gopkg.git shellcheck];
          setup = ''
            git init --quiet
            git add --all
          '';
        };

        lint-sql = mkBuildStep {
          name = "lint-sql";
          command = "./dev-scripts/lint-sql";
          src = gopkg.lib.fileset.toSource {
            root = ./.;
            fileset = gopkg.lib.fileset.unions [
              ./dev-scripts/lint-sql
              ./.sqlfluffignore
              (gopkg.lib.fileset.fileFilter
                (file: file.hasExt "sql")
                ./store/sqlite/migrations)
            ];
          };
          extraInputs = [sqlfluff];
        };

        check-go-formatting = mkBuildStep {
          name = "check-go-formatting";
          command = "./dev-scripts/check-go-formatting";
          src = gopkg.lib.fileset.toSource {
            root = ./.;
            fileset =
              gopkg.lib.fileset.intersection
              (gopkg.lib.fileset.gitTracked ./.)
              (gopkg.lib.fileset.unions [
                ./dev-scripts/check-go-formatting
                (gopkg.lib.fileset.fileFilter (file: file.hasExt "go") ./.)
              ]);
          };
          extraInputs = [go];
        };

        check-go-test-packages = mkBuildStep {
          name = "check-go-test-packages";
          command = "./dev-scripts/check-go-test-packages";
          src = gopkg.lib.fileset.toSource {
            root = ./.;
            fileset =
              gopkg.lib.fileset.intersection
              (gopkg.lib.fileset.gitTracked ./.)
              (gopkg.lib.fileset.unions [
                ./dev-scripts/check-go-test-packages
                (gopkg.lib.fileset.fileFilter
                  (file: gopkg.lib.hasSuffix "_test.go" file.name)
                  ./.)
              ]);
          };
          extraInputs = [gopkg.git gopkg.gawk];
          setup = ''
            git init --quiet
            git add --all
          '';
        };

        check-trailing-newline = mkBuildStep {
          name = "check-trailing-newline";
          command = "./dev-scripts/check-trailing-newline";
          src = gopkg.lib.fileset.toSource {
            root = ./.;
            fileset = gopkg.lib.fileset.gitTracked ./.;
          };
          extraInputs = [gopkg.git gopkg.coreutils gopkg.findutils gopkg.gnugrep];
          setup = ''
            git init --quiet
            git add --all
          '';
        };

        check-trailing-whitespace = mkBuildStep {
          name = "check-trailing-whitespace";
          command = "./dev-scripts/check-trailing-whitespace";
          src = gopkg.lib.fileset.toSource {
            root = ./.;
            fileset = gopkg.lib.fileset.gitTracked ./.;
          };
          extraInputs = [gopkg.git gopkg.gnugrep];
          setup = ''
            git init --quiet
            git add --all
          '';
        };

        e2e-tests = let
          playwrightBrowsers = playwright-nixpkgs.legacyPackages.${system}.playwright-driver.browsers;
          fontsConf = nodepkgs.makeFontsConf {
            fontDirectories = [nodepkgs.dejavu_fonts];
          };
        in
          buildNpmPackage {
            pname = "picoshare-e2e-tests";
            version = "0.0.1";
            src = gopkg.lib.fileset.toSource {
              root = ./.;
              fileset =
                gopkg.lib.fileset.intersection
                (gopkg.lib.fileset.gitTracked ./.)
                (gopkg.lib.fileset.unions [
                  ./dev-scripts/run-e2e-tests
                  ./e2e
                  ./package.json
                  ./package-lock.json
                ]);
            };
            inherit npmDeps;
            npmInstallFlags = ["--ignore-scripts"];
            dontNpmBuild = true;
            nativeBuildInputs = [nodejs playwrightBrowsers appPackageDev];
            doCheck = true;
            checkPhase = ''
              export HOME="$PWD/.home"
              mkdir -p "$HOME"
              export CI=1
              export PLAYWRIGHT_BROWSERS_PATH=${playwrightBrowsers}
              export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
              export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true

              # Configure fonts for headless browser rendering.
              export FONTCONFIG_FILE=${fontsConf}

              mkdir -p ./bin
              cp ${appPackageDev}/bin/${appNameDev} ./bin/${appNameDev}

              patchShebangs ./dev-scripts
              ./dev-scripts/run-e2e-tests --skip-build --project=chromium
            '';
            installPhase = ''
              mkdir -p "$out"
              printf 'e2e-tests passed\n' > "$out/result"
            '';
          };
      };

      checks.all = gopkg.linkFarm "all-flake-targets" self.packages.${system};

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
            gci
            go-tools
            go-critic
            errcheck
            go
            sqlite
            nodejs
            shellcheck
            sqlfluff
            flyctl
            litestream
            air
          ];

          shellHook = ''
            # Ignore user Go settings so Nix's pinned toolchain is authoritative.
            export GOENV='off'
            export GOTOOLCHAIN='local'
            unset GOROOT

            # Isolate `go install`ed binaries per checkout, while sharing Go's
            # default, content-addressed module cache across projects.
            export GOBIN="$PWD/bin/.go"
            export PATH="$GOBIN:$PATH"

            # Restore the exact lockfile dependency set when manifests change.
            if [ -f package.json ]; then
              if [ ! -d node_modules ] \
                || [ package.json -nt node_modules ] \
                || [ package-lock.json -nt node_modules ]; then
                echo "Installing npm packages..."
                if CI=true npm ci; then
                  touch node_modules
                else
                  echo "Failed to install npm packages" >&2
                  exit 1
                fi
              fi
            fi

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
