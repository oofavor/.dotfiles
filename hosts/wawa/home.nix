{ config, pkgs, ... }:

{
  imports = [
    ../../home/core
    ../../home/desktop/sway
    ../../home/desktop/waybar
    ../../home/gui
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
