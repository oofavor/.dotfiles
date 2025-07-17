{ pkgs, config, ... }:
{
  home.packages = with pkgs; [
    networkmanagerapplet
    pwvucontrol
  ];

  programs.waybar = {
    enable = true;
    style = builtins.readFile ./style.css;
    settings = {
      mainBar = ./config.nix;
    };
  };
}
