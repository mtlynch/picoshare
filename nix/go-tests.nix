{
  pkgs,
  go,
  sqlite,
}: let
  buildGoModule = pkgs.buildGoModule.override {inherit go;};
  # These tools load packages using the same Go version as the tests.
  go-critic = (pkgs.go-critic.override {inherit buildGoModule;}).overrideAttrs {
    subPackages = ["cmd/go-critic"];
  };
  staticcheck = pkgs.go-tools.override {inherit buildGoModule;};
  errcheck = pkgs.errcheck.override {inherit buildGoModule;};
in
  assert pkgs.gci.version == "0.13.7";
  assert go-critic.version == "0.14.3";
  # Staticcheck's 2026.1 release is Go module version v0.7.0.
  assert staticcheck.version == "2026.1";
  assert errcheck.version == "1.10.0";
    buildGoModule {
      pname = "picoshare-go-tests";
      version = "0.0.1";
      src = pkgs.lib.fileset.toSource {
        root = ../.;
        fileset =
          pkgs.lib.fileset.intersection
          (pkgs.lib.fileset.gitTracked ../.)
          (pkgs.lib.fileset.unions [
            ../dev-scripts/run-go-tests
            ../go.mod
            ../go.sum
            ../handlers/templates
            ../handlers/static
            ../store/sqlite/migrations
            (pkgs.lib.fileset.fileFilter (file: file.hasExt "go") ../.)
          ]);
      };
      vendorHash = "sha256-Q2PtwnlWl71XOv0RUM4oWNknGPeyjBkGvO7dLQ93jlY=";
      nativeBuildInputs = [pkgs.git pkgs.gci go-critic staticcheck errcheck sqlite];
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
    }
