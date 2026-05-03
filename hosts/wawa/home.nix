{ pkgs, ... }:
{
  imports = [ ../../home ];

  home.packages = with pkgs; [
    powertop
    brightnessctl
  ];

  ofavor.desktop.enable = true;
  ofavor.dev.langs = [
    "rust"
    "node"
    "go"
  ];
}
