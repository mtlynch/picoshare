{
  pkgs,
  binaries,
  images,
}:
pkgs.writeShellApplication {
  name = "publish-release";
  runtimeInputs = with pkgs; [bash coreutils gzip gnutar git gh crane jq];
  text = ''
    exec bash ${../dev-scripts/publish-release} \
      ${binaries} ${images} ${../dev-scripts/package-binaries}
  '';
}
