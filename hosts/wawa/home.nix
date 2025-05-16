{
  config,
  pkgs,
  lib,
  ...
}:
with lib;
{
  imports = [ ../../home ];

  home.username = "ofavor";
  home.homeDirectory = "/home/ofavor";

  home.stateVersion = "24.11";

  home.packages = with pkgs; [
    powertop
    brightnessctl
  ];

  programs.home-manager.enable = true;

}
