{ config, ... }:
let
  ghosttyPath = "${config.home.homeDirectory}/.dotfiles/home/gui/terminals/ghostty";
in
{
  xdg.configFile."ghostty".source = config.lib.file.mkOutOfStoreSymlink ghosttyPath;
  programs.ghostty = {
    enable = true;
    # settings = {
    #   theme = "tokyonight";
    #   font-family = "Cascadia Code NF";
    #   # no ligatures!!!
    #   font-feature = "-calt, -liga, -dlig";
    # };
  };
}
