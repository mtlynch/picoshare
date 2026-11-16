{
  systems = ["x86_64-linux"];
  deploy = {
    fly = {
      package = "packages.x86_64-linux.deploy-to-fly";
      system = "x86_64-linux";
      branches = ":default";
      in-repo = true;
      secrets = ["FLY_API_TOKEN"];
    };
    release = {
      package = "packages.x86_64-linux.publish-release";
      system = "x86_64-linux";
      branches = "v*";
      in-repo = true;
      secrets = ["GH_TOKEN" "DOCKERHUB_USERNAME" "DOCKERHUB_ACCESS_TOKEN"];
    };
  };
}
