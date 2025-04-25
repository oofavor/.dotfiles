{ config, pkgs, ... }:

{
  imports = [ ./../../home/base ./../../home/gui ./../../home/hypr ];

  home.username = "ofavor";
  home.homeDirectory = "/home/ofavor";

  home.stateVersion = "24.11";

  programs.home-manager.enable = true;
}
