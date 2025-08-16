{
  pkgs,
  config,
  lib,
  ...
}:
{
  imports = [
    ./../hypr/hyprpaper.nix
    ./../hypr/hypridle.nix
  ];

  home.packages = with pkgs; [ xwayland-satellite ];
  programs.niri.settings = {
    outputs."PNP(AOC) 24G1WG4 0x00000E96".mode = {
      width = 1920;
      height = 1080;
      refresh = 144.001;
    };

    outputs."BOE 0x0AF2 Unknown" = {
      scale = 2;
      mode = {
        width = 2560;
        height = 1600;
        refresh = 120.000;
      };
    };

    workspaces."01".name = "terminal";
    workspaces."02".name = "browser";
    workspaces."03".name = "misc";

    input = {
      keyboard = {
        xkb = {
          layout = "us,ru";
          options = "ctrl:nocaps,grp:alt_shift_toggle";
        };
      };
      touchpad = {
        tap = true;
        natural-scroll = true;
        accel-speed = -0.2;
        scroll-factor = 0.6;
      };
    };

    environment = {
      QT_QPA_PLATFORM = "wayland";
      DISPLAY = ":0";
    };

    spawn-at-startup = [
      { command = [ "xwayland-satellite" ]; }
      { command = [ "hyprpaper" ]; }
      { command = [ "dunst" ]; }
    ];

    binds = with config.lib.niri.actions; {
      "Mod+1".action = focus-workspace "terminal";
      "Mod+2".action = focus-workspace "browser";
      "Mod+3".action = focus-workspace "misc";

      # TODO: uncomment when niri flake fixes this
      # "Mod+Shift+1".action = move-column-to-workspace 1;
      # "Mod+Shift+2".action = move-column-to-workspace 2;
      # "Mod+Shift+3".action = move-column-to-workspace 3;

      "Mod+Shift+1".action = spawn "niri" "msg" "action" "move-column-to-workspace" "terminal";
      "Mod+Shift+2".action = spawn "niri" "msg" "action" "move-column-to-workspace" "browser";
      "Mod+Shift+3".action = spawn "niri" "msg" "action" "move-column-to-workspace" "misc";

      "Mod+h".action = focus-column-left;
      "Mod+l".action = focus-column-right;
      "Mod+Shift+h".action = move-column-left;
      "Mod+Shift+l".action = move-column-right;
      "Mod+v".action = toggle-window-floating;

      "Mod+minus".action = set-column-width "-10%";
      "Mod+equal".action = set-column-width "+10%";
      "Mod+s".action = maximize-column;

      "Mod+r".action = spawn "rofi" "-show" "drun";
      "Mod+b".action = spawn "dunstctl" "history-pop";
      "Mod+q".action = spawn "ghostty";

      "Mod+c".action = close-window;
      "Mod+Shift+m".action = quit;
    };
  };
}
