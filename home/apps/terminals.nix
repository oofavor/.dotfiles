{ pkgs, lib, config, ... }:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    home.sessionVariables = {
      TERMINAL = "ghostty";
    };

    programs.ghostty = {
      enable = true;
      settings = {
        # theme = "tokyonight";
        font-family = "CaskaydiaCove";
        font-size = 16.3;
        # font-family = "Monaspace Neon";
        # no ligatures!!!
        font-feature = "-calt, -liga, -dlig";
      };
    };
  };
}
