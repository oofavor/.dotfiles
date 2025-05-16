{ config, pkgs, ... }:

{
  imports = [ ../../home ];

  home.username = "ofavor";
  home.homeDirectory = "/home/ofavor";

  home.stateVersion = "24.11";

  programs.home-manager.enable = true;

  ofavor.games.enable = true;
}
