{
  pkgs,
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    home.sessionVariables = {
      TERMINAL = "ghostty";
    };

    programs.ghostty = {
      enable = true;
      settings = {
        # theme = "tokyonight";
        # no ligatures!!!
        font-feature = "-calt, -liga, -dlig";
        font-size = 16.3;
        gtk-single-instance = true;
        linux-cgroup = "never";
        window-padding-y = 0;
        window-padding-color = "extend-always";
      };
    };
  };
}
