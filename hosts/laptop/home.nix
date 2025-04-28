{ config, pkgs, ... }:

{
  imports = [
    ./../../home/base
    ./../../home/gui
    ./../../home/wm/sway
    ./../../home/dev
  ];

  home.username = "ofavor";
  home.homeDirectory = "/home/ofavor";

  home.stateVersion = "24.11";

  home.packages = with pkgs; [
    powertop
    brightnessctl
  ];

  programs.home-manager.enable = true;
}
