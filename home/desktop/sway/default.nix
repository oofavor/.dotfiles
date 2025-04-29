{ pkgs, config, ... }:
{
  wayland.windowManager.sway = {
    enable = true;
    wrapperFeatures.gtk = true; # Fixes common issues with GTK 3 apps
    config = rec {
      modifier = "Mod4";
      terminal = "ghostty";
      startup = [
        { command = "firefox"; }
      ];
      input = {
        "1267:12856:ELAN2310:00_04F3:3238_Touchpad" = {
          natural_scroll = "enabled";
          tap = "enabled";
        };
      };
      window = {
        titlebar = false;
      };
    };
  };

  # xdg.configFile."waybar".source = config.lib.file.mkOutOfStoreSymlink ./waybar;
  programs.waybar = {
    enable = true;
    systemd.enable = true;
    # style = builtins.readFile ./waybar/styles.css;
  };
}
