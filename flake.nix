{
  description = "Dev environment for WhatGotDone";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/release-23.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; };

      pkgs_for_sqlfluff = import (builtins.fetchTarball {
        # 1.2.1 release
        url = "https://github.com/NixOS/nixpkgs/archive/7cf5ccf1cdb2ba5f08f0ac29fc3d04b0b59a07e4.tar.gz";
        sha256 = "0wfaqjpi7bip86r2piqigqna1fx3m1d9riak4l3rm54lyjxprlpi";
      }) {inherit system; };

      pkgs_for_go = import (builtins.fetchTarball {
        # 1.19.6 release
        url = "https://github.com/NixOS/nixpkgs/archive/6adf48f53d819a7b6e15672817fa1e78e5f4e84f.tar.gz";
        sha256 = "0p7m72ipxyya5nn2p8q6h8njk0qk0jhmf6sbfdiv4sh05mbndj4q";
      }) {inherit system; };

      pkgs_for_nodejs = import (builtins.fetchTarball {
        # 20.6.1 release
        url = "https://github.com/NixOS/nixpkgs/archive/78058d810644f5ed276804ce7ea9e82d92bee293.tar.gz";
        sha256 = "1k0bsy98ybpb05ddlhj2w0xbzinnidl2cdl9ifq96lvi04xvns4d";
      }) {inherit system; };
    in
    {
      devShells.default = pkgs_for_go.mkShell.override { stdenv = pkgs_for_go.pkgsStatic.stdenv; } {
        packages = with pkgs; [
          gopls
          gotools
          pkgs_for_go.go_1_19
          pkgs_for_nodejs.nodejs_20
          pkgs_for_sqlfluff.sqlfluff
        ];

        shellHook = ''
          echo "node" "$(node --version)"
          echo "npm" "$(npm --version)"
          go version
          sqlfluff --version
        '';
      };
    });
}
