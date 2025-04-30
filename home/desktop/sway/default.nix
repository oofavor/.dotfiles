{ pkgs, config, ... }:
let
  swayPath = "${config.home.homeDirectory}/.dotfiles/home/desktop/sway/sway";
in
{
  xdg.configFile."sway".source = config.lib.file.mkOutOfStoreSymlink swayPath;

  services.gnome-keyring.enable = true;

  # wayland.windowManager.sway = {
  #   enable = true;
  #
  # };
}
