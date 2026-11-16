{
  pkgs,
  flyctl,
  image,
}: let
  skopeoPolicy = pkgs.writeText "containers-policy.json" (builtins.toJSON {
    default = [
      {type = "reject";}
    ];
    transports.docker-archive."" = [
      # Trust only the local archive; reject every other image source.
      {type = "insecureAcceptAnything";}
    ];
  });
  streamDockerImage = pkgs.writeShellScriptBin "stream-docker-image" ''
    exec ${image}
  '';
in
  pkgs.writeShellApplication {
    name = "deploy-to-fly";
    runtimeInputs = [
      pkgs.bash
      flyctl
      pkgs.skopeo
      pkgs.git
      pkgs.coreutils
      streamDockerImage
    ];
    text = ''
      exec bash ${../dev-scripts/deploy-to-fly} ${skopeoPolicy}
    '';
    meta.mainProgram = "deploy-to-fly";
  }
