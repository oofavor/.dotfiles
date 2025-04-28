{ pkgs, ... }:
{
  wayland.windowManager.sway = {
    enable = true;
    wrapperFeatures.gtk = true; # Fixes common issues with GTK 3 apps
    config = rec {
      modifier = "Mod4";
      # Use kitty as default terminal
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
}
