{ config, pkgs, ... }:

{

  imports = [
    ./../../home/base
    ./../../home/dev
    ./../../home/gui
    ./../../home/wm/hypr
  ];

  home.username = "ofavor";
  home.homeDirectory = "/home/ofavor";

  home.stateVersion = "24.11";

  programs.home-manager.enable = true;
}
