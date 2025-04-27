{ config, pkgs, ... }:

{
  imports = [ ./../../home/base ./../../home/gui ./../../home/hypr ./../../home/dev];

  home.username = "ofavor";
  home.homeDirectory = "/home/ofavor";

  home.stateVersion = "24.11";

  home.packages = with pkgs; [ powertop brightnessctl upower ];

  programs.home-manager.enable = true;
}
