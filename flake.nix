{
  description = "Dev environment for PicoShare";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";

    # Use https://www.nixhub.io/ to find the right nixpkgs commit for the
    # specific package version we want.

    # 1.26.1 release
    go-nixpkgs.url = "github:NixOS/nixpkgs/e607cb5360ff1234862ac9f8839522becb853bb9";

    # 3.44.2 release
    sqlite-nixpkgs.url = "github:NixOS/nixpkgs/5ad9903c16126a7d949101687af0aa589b1d7d3d";

    # 24.11.1 release
    nodejs-nixpkgs.url = "github:NixOS/nixpkgs/af84f9d270d404c17699522fab95bbf928a2d92f";

    # 0.9.0 release
    shellcheck-nixpkgs.url = "github:NixOS/nixpkgs/8b5ab8341e33322e5b66fb46ce23d724050f6606";

    # 3.1.1 release
    sqlfluff-nixpkgs.url = "github:NixOS/nixpkgs/5629520edecb69630a3f4d17d3d33fc96c13f6fe";

    # 1.59.1
    playwright-nixpkgs.url = "github:NixOS/nixpkgs/7aaa00e7cc9be6c316cb5f6617bd740dd435c59d";

    # 0.4.59 release
    flyctl-nixpkgs.url = "github:NixOS/nixpkgs/5a722a7155bfc9fbe657f28d26b71860d95324bc";

    # 0.3.13 release
    litestream-nixpkgs.url = "github:NixOS/nixpkgs/a343533bccc62400e8a9560423486a3b6c11a23b";

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
      };
      sqlite = sqlite-nixpkgs.legacyPackages.${system}.sqlite;
      nodepkgs = nodejs-nixpkgs.legacyPackages.${system};
      nodejs = nodepkgs.nodejs_24;
      buildNpmPackage = nodepkgs.buildNpmPackage.override {inherit nodejs;};
      shellcheck = shellcheck-nixpkgs.legacyPackages.${system}.shellcheck;
      sqlfluff = sqlfluff-nixpkgs.legacyPackages.${system}.sqlfluff;
      flyctl = flyctl-nixpkgs.legacyPackages.${system}.flyctl;
      litestream = litestream-nixpkgs.legacyPackages.${system}.litestream;
      air = air-nixpkgs.legacyPackages.${system}.air;

      # Backend sources omit tests and their test-only dependencies.
      goVendorHash = "sha256-autscgwM+dSiA9L15kX6BQkA8eAr7+Guxg6ypr2a4MI=";

      npmDepsHash = "sha256-vlpvjZBjSn+dx4s+mdp/2kI4TbXmpP+kWYwjwRLhBxE=";

      backendSrc = gopkg.lib.fileset.toSource {
        root = ./.;
        fileset =
          gopkg.lib.fileset.intersection
          (gopkg.lib.fileset.gitTracked ./.)
          (gopkg.lib.fileset.unions [
            ./go.mod
            ./go.sum
            (gopkg.lib.fileset.fileFilter
              (file: file.hasExt "go" && !(gopkg.lib.hasSuffix "_test.go" file.name))
              ./.)
            ./handlers/static
            ./handlers/templates
            ./store/sqlite/migrations
          ]);
      };

      release = let
        inherit (gopkg) lib;
        litestreamVersion = litestream-nixpkgs.legacyPackages.${system}.litestream.version;
        targets = {
          linux-amd64 = {
            packages = gopkg.pkgsCross.gnu64;
            architecture = "amd64";
            litestreamArchitecture = "amd64";
            litestreamHash = "sha256-63Wj3lyrA4dc2un19TnmrtrdZmBwA9mx56kHeUiBi6A=";
          };
          linux-arm64 = {
            packages = gopkg.pkgsCross.aarch64-multiplatform;
            architecture = "arm64";
            litestreamArchitecture = "arm64";
            litestreamHash = "sha256-lYX1pQhRa9Zq8rI3a6tN4lal744rc+x2BVnmeWKPLVk=";
          };
          linux-armv7 = {
            packages = gopkg.pkgsCross.armv7l-hf-multiplatform;
            architecture = "arm";
            litestreamArchitecture = "arm7";
            litestreamHash = "sha256-Il8tDsQx3TLTjuR+OUHrcv4DFqr/2UEYW6MPfG3iGGY=";
          };
        };
        mkBackend = target: let
          targetBuildGoModule = gopkg.buildGoModule.override {
            inherit go;
          };
        in
          targetBuildGoModule {
            pname = "picoshare";
            version = "0.0.1";
            src = backendSrc;
            vendorHash = goVendorHash;
            subPackages = ["cmd/picoshare"];
            env =
              {
                CGO_ENABLED = "0";
                GOOS = "linux";
                GOARCH = target.architecture;
              }
              // lib.optionalAttrs (target.architecture == "arm") {GOARM = "7";};
            tags = ["netgo"];
            ldflags = ["-w"];
            doCheck = false;
          };
        binaries = lib.mapAttrs (_: mkBackend) targets;
        mkImage = platform: target: let
          litestreamArchive = gopkg.fetchurl {
            url = "https://github.com/benbjohnson/litestream/releases/download/v${litestreamVersion}/litestream-v${litestreamVersion}-linux-${target.litestreamArchitecture}.tar.gz";
            hash = target.litestreamHash;
          };
          runtime = gopkg.buildEnv {
            name = "picoshare-runtime-${platform}";
            paths = [target.packages.bash target.packages.coreutils target.packages.cacert];
            pathsToLink = ["/bin" "/etc/ssl/certs"];
          };
        in
          gopkg.dockerTools.streamLayeredImage {
            name = "picoshare";
            tag = platform;
            inherit (target) architecture;
            # Preserve the root account needed by Fly's SSH console.
            contents = [runtime gopkg.dockerTools.fakeNss];
            extraCommands = ''
              mkdir -p app etc usr/bin data tmp
              chmod 1777 tmp
              cp ${binaries.${platform}}/bin/picoshare app/picoshare
              tar --extract --gzip --file=${litestreamArchive} --directory=app litestream
              cp ${./docker-entrypoint} app/docker-entrypoint
              chmod +x app/docker-entrypoint app/litestream
              cp ${./litestream.yml} etc/litestream.yml
              cp ${./LICENSE} app/LICENSE
              ln -s /bin/env usr/bin/env
            '';
            config = {
              WorkingDir = "/app";
              Entrypoint = ["/app/docker-entrypoint"];
              Cmd = ["-db" "/data/store.db"];
              Env = ["PATH=/bin:/usr/bin" "LITESTREAM_RETENTION=72h" "LITESTREAM_PATH=db"];
            };
          };
        images = lib.mapAttrs mkImage targets;
        nativePlatform = "linux-${gopkg.stdenv.hostPlatform.go.GOARCH}";
      in {
        backend = binaries.${nativePlatform};
        docker-image = images.${nativePlatform};
        release-binaries = gopkg.runCommand "picoshare-release-binaries" {} (
          lib.concatStringsSep "\n" (lib.mapAttrsToList (platform: binary: ''
              mkdir -p "$out/${platform}"
              cp ${binary}/bin/picoshare "$out/${platform}/picoshare"
            '')
            binaries)
        );
        release-images = gopkg.runCommand "picoshare-release-images" {} (
          ''mkdir -p "$out"''
          + "\n"
          + lib.concatStringsSep "\n" (lib.mapAttrsToList (platform: image: ''
              ln -s ${image} "$out/${platform}"
            '')
            images)
        );
      };

      goTests = let
        testBuildGoModule = gopkg.buildGoModule.override {inherit go;};
        # These tools load packages using the same Go version as the tests.
        go-critic = (gopkg.go-critic.override {buildGoModule = testBuildGoModule;}).overrideAttrs {
          subPackages = ["cmd/go-critic"];
        };
        staticcheck = gopkg.go-tools.override {buildGoModule = testBuildGoModule;};
        errcheck = gopkg.errcheck.override {buildGoModule = testBuildGoModule;};
      in
        assert gopkg.gci.version == "0.13.7";
        assert go-critic.version == "0.14.3";
        # Staticcheck's 2026.1 release is Go module version v0.7.0.
        assert staticcheck.version == "2026.1";
        assert errcheck.version == "1.10.0";
          testBuildGoModule {
            pname = "picoshare-go-tests";
            version = "0.0.1";
            src = gopkg.lib.fileset.toSource {
              root = ./.;
              fileset =
                gopkg.lib.fileset.intersection
                (gopkg.lib.fileset.gitTracked ./.)
                (gopkg.lib.fileset.unions [
                  ./dev-scripts/run-go-tests
                  ./go.mod
                  ./go.sum
                  ./handlers/templates
                  ./handlers/static
                  ./store/sqlite/migrations
                  (gopkg.lib.fileset.fileFilter (file: file.hasExt "go") ./.)
                ]);
            };
            vendorHash = "sha256-Q2PtwnlWl71XOv0RUM4oWNknGPeyjBkGvO7dLQ93jlY=";
            nativeBuildInputs = [gopkg.git gopkg.gci go-critic staticcheck errcheck sqlite];
            env.CGO_ENABLED = "0";
            dontBuild = true;
            doCheck = true;
            checkPhase = ''
              runHook preCheck
              export HOME="$TMPDIR/home"
              mkdir -p "$HOME"
              patchShebangs ./dev-scripts
              git init --quiet
              git add --all
              ./dev-scripts/run-go-tests --full
              runHook postCheck
            '';
            installPhase = ''
              runHook preInstall
              mkdir -p "$out"
              cp .coverage.out .coverage.html "$out/"
              runHook postInstall
            '';
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

      backend-dev = buildGoModule {
        pname = "picoshare-dev";
        version = "0.0.1";
        src = backendSrc;
        vendorHash = goVendorHash;
        subPackages = ["cmd/picoshare"];
        env.CGO_ENABLED = "0";
        tags = ["netgo" "dev"];
        ldflags = ["-w"];
        postInstall = ''
          mv "$out/bin/picoshare" "$out/bin/picoshare-dev"
        '';
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
          patchShebangs ./dev-scripts
          git init --quiet
          git add --all
          ./dev-scripts/build-frontend
          runHook postCheck
        '';
        installPhase = ''
          mkdir -p "$out"
          printf 'frontend-check passed\n' > "$out/result"
        '';
      };
    in {
      packages =
        {
          inherit backend-dev frontend-check;

          go-tests = goTests;

          check-ci-workflows = mkBuildStep {
            name = "check-ci-workflows";
            command = "python3 ./dev-scripts/tests/test-ci-workflows.py";
            src = gopkg.lib.fileset.toSource {
              root = ./.;
              fileset = gopkg.lib.fileset.unions [
                ./dev-scripts/build-all-flake-targets
                ./dev-scripts/upload-to-nixci-cache
                ./dev-scripts/package-binaries
                ./dev-scripts/publish-release
                ./dev-scripts/deploy-to-fly
                ./dev-scripts/tests/test-ci-workflows.py
              ];
            };
            extraInputs = [gopkg.python3 gopkg.jq gopkg.git gopkg.gnutar gopkg.gzip];
            extraAttrs = gopkg.lib.optionalAttrs gopkg.stdenv.isLinux {
              env.LOCALE_ARCHIVE = "${gopkg.glibcLocales}/lib/locale/locale-archive";
            };
          };

          check-bash = mkBuildStep {
            name = "check-bash";
            command = "./dev-scripts/check-bash";
            src = gopkg.lib.fileset.toSource {
              root = ./.;
              fileset = gopkg.lib.fileset.gitTracked ./.;
            };
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
              nativeBuildInputs = [nodejs playwrightBrowsers backend-dev];
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
                cp ${backend-dev}/bin/picoshare-dev ./bin/picoshare-dev

                patchShebangs ./dev-scripts
                ./dev-scripts/run-e2e-tests --skip-build
              '';
              installPhase = ''
                mkdir -p "$out"
                printf 'e2e-tests passed\n' > "$out/result"
                cp --recursive playwright-report "$out/"
              '';
            };
        }
        // gopkg.lib.optionalAttrs gopkg.stdenv.isLinux {
          inherit (release) backend release-binaries docker-image release-images;

          deploy-to-fly = let
            skopeoPolicy = gopkg.writeText "containers-policy.json" (builtins.toJSON {
              default = [{type = "reject";}];
              transports.docker-archive."" = [
                # Trust only the local archive; reject every other image source.
                {type = "insecureAcceptAnything";}
              ];
            });
            streamDockerImage = gopkg.writeShellScriptBin "stream-docker-image" ''
              exec ${release.docker-image}
            '';
          in
            gopkg.writeShellApplication {
              name = "deploy-to-fly";
              runtimeInputs = [gopkg.bash flyctl gopkg.skopeo gopkg.git gopkg.coreutils streamDockerImage];
              text = ''
                exec bash ${./dev-scripts/deploy-to-fly} ${skopeoPolicy}
              '';
              meta.mainProgram = "deploy-to-fly";
            };

          publish-release = gopkg.writeShellApplication {
            name = "publish-release";
            runtimeInputs = with gopkg; [bash coreutils gzip gnutar git gh crane jq];
            text = ''
              exec bash ${./dev-scripts/publish-release} \
                ${release.release-binaries} ${release.release-images} ${./dev-scripts/package-binaries}
            '';
          };
        };

      checks = {
        all = gopkg.linkFarm "all-flake-targets" self.packages.${system};
        inherit frontend-check;
        inherit
          (self.packages.${system})
          check-go-formatting
          check-go-test-packages
          check-trailing-newline
          check-trailing-whitespace
          lint-sql
          ;
      };

      devShells.default = go-nixpkgs.legacyPackages.${system}.mkShell {
        packages = [
          go-nixpkgs.legacyPackages.${system}.gotools
          go-nixpkgs.legacyPackages.${system}.gopls
          go-nixpkgs.legacyPackages.${system}.go-outline
          go-nixpkgs.legacyPackages.${system}.gopkgs
          go-nixpkgs.legacyPackages.${system}.gocode-gomod
          go-nixpkgs.legacyPackages.${system}.godef
          go-nixpkgs.legacyPackages.${system}.golint
          go-nixpkgs.legacyPackages.${system}.gci
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
          # Ignore user-level Go settings and keep installed tools local to
          # this checkout.
          export GOENV=off
          export GOTOOLCHAIN=local
          export GOBIN="$PWD/bin"
          export PATH="$GOBIN:$PATH"

          # Restore the exact lockfile dependency set when manifests change.
          if [ -f package.json ]; then
            if [ ! -d node_modules ] \
              || [ package.json -nt node_modules ] \
              || [ package-lock.json -nt node_modules ]; then
              echo "Installing npm packages..."
              if npm ci; then
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
