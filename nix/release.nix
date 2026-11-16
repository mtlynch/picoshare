{
  pkgs,
  src,
  vendorHash,
  litestreamPkgs,
}: let
  inherit (pkgs) lib;
  litestreamVersion = litestreamPkgs.legacyPackages.${pkgs.stdenv.buildPlatform.system}.litestream.version;
  targets = {
    linux-amd64 = {
      packages = pkgs.pkgsCross.gnu64;
      architecture = "amd64";
      litestreamArchitecture = "amd64";
      litestreamHash = "sha256-63Wj3lyrA4dc2un19TnmrtrdZmBwA9mx56kHeUiBi6A=";
    };
    linux-arm64 = {
      packages = pkgs.pkgsCross.aarch64-multiplatform;
      architecture = "arm64";
      litestreamArchitecture = "arm64";
      litestreamHash = "sha256-lYX1pQhRa9Zq8rI3a6tN4lal744rc+x2BVnmeWKPLVk=";
    };
    linux-armv7 = {
      packages = pkgs.pkgsCross.armv7l-hf-multiplatform;
      architecture = "arm";
      litestreamArchitecture = "arm7";
      litestreamHash = "sha256-Il8tDsQx3TLTjuR+OUHrcv4DFqr/2UEYW6MPfG3iGGY=";
    };
  };

  mkBackend = target: let
    buildGoModule = pkgs.buildGoModule.override {
      go = pkgs.go_1_26;
    };
  in
    buildGoModule {
      pname = "picoshare";
      version = "0.0.1";
      inherit src vendorHash;
      subPackages = ["cmd/picoshare"];
      env =
        {
          CGO_ENABLED = "0";
          GOOS = "linux";
          GOARCH = target.architecture;
        }
        // lib.optionalAttrs (target.architecture == "arm") {
          GOARM = "7";
        };
      tags = ["netgo"];
      ldflags = ["-w"];
      doCheck = false;
    };
  binaries = lib.mapAttrs (_: mkBackend) targets;

  mkImage = platform: target: let
    litestreamArchive = pkgs.fetchurl {
      url = "https://github.com/benbjohnson/litestream/releases/download/v${litestreamVersion}/litestream-v${litestreamVersion}-linux-${target.litestreamArchitecture}.tar.gz";
      hash = target.litestreamHash;
    };
    runtime = pkgs.buildEnv {
      name = "picoshare-runtime-${platform}";
      paths = [target.packages.bash target.packages.coreutils target.packages.cacert];
      pathsToLink = ["/bin" "/etc/ssl/certs"];
    };
  in
    pkgs.dockerTools.streamLayeredImage {
      name = "picoshare";
      tag = platform;
      inherit (target) architecture;
      # Preserve the root account needed by Fly's SSH console.
      contents = [runtime pkgs.dockerTools.fakeNss];
      extraCommands = ''
        mkdir -p app etc usr/bin data tmp
        chmod 1777 tmp
        cp ${binaries.${platform}}/bin/picoshare app/picoshare
        tar --extract --gzip --file=${litestreamArchive} --directory=app litestream
        cp ${../docker-entrypoint} app/docker-entrypoint
        chmod +x app/docker-entrypoint app/litestream
        cp ${../litestream.yml} etc/litestream.yml
        cp ${../LICENSE} app/LICENSE
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
  nativePlatform = "linux-${pkgs.stdenv.hostPlatform.go.GOARCH}";
in {
  backend = binaries.${nativePlatform};
  docker-image = images.${nativePlatform};
  release-binaries = pkgs.runCommand "picoshare-release-binaries" {} (
    lib.concatStringsSep "\n" (lib.mapAttrsToList (platform: binary: ''
        mkdir -p "$out/${platform}"
        cp ${binary}/bin/picoshare "$out/${platform}/picoshare"
      '')
      binaries)
  );
  release-images = pkgs.runCommand "picoshare-release-images" {} (
    ''mkdir -p "$out"''
    + "\n"
    + lib.concatStringsSep "\n" (lib.mapAttrsToList (platform: image: ''
        ln -s ${image} "$out/${platform}"
      '')
      images)
  );
}
