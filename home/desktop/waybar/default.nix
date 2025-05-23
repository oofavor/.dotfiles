{ pkgs, config, ... }:
let
  waybarPath = "${config.home.homeDirectory}/.dotfiles/home/desktop/waybar/config";
in
{
  xdg.configFile."waybar".source = config.lib.file.mkOutOfStoreSymlink waybarPath;

  home.packages = with pkgs; [
    networkmanagerapplet
    pwvucontrol
  ];

  programs.waybar = {
    enable = true;
  };
}
