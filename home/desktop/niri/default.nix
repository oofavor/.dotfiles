{
  pkgs,
  config,
  lib,
  ...
}:
let
  wallpaper = ../../../assets/background.jpg;
  lockCommand = "${pkgs.swaylock}/bin/swaylock -f";
in
{
  imports = [
    ./waybar.nix
  ];

  config = lib.mkIf config.ofavor.desktop.enable {

    home.packages = with pkgs; [
      brightnessctl
      swaybg
      swayidle
      xwayland-satellite
    ];

    programs.swaylock = {
      enable = true;
      settings = {
        image = wallpaper;
        scaling = "fill";
        show-failed-attempts = true;
      };
    };

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
        {
          command = [
            "swaybg"
            "-i"
            "${wallpaper}"
            "-m"
            "fill"
          ];
        }
        {
          command = [
            "swayidle"
            "-w"
            "timeout"
            "150"
            "${pkgs.brightnessctl}/bin/brightnessctl -s set 10"
            "resume"
            "${pkgs.brightnessctl}/bin/brightnessctl -r"
            "timeout"
            "300"
            lockCommand
            "timeout"
            "330"
            "niri msg action power-off-monitors"
            "resume"
            "niri msg action power-on-monitors"
            "before-sleep"
            lockCommand
            "lock"
            lockCommand
          ];
        }
        { command = [ "waybar" ]; }
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

        XF86MonBrightnessUp = {
          allow-when-locked = true;
          action.spawn = [
            "brightnessctl"
            "set"
            "+10%"
          ];
        };

        XF86MonBrightnessDown = {
          allow-when-locked = true;
          action.spawn = [
            "brightnessctl"
            "set"
            "10%-"
          ];
        };

        XF86AudioRaiseVolume = {
          allow-when-locked = true;
          action.spawn = [
            "wpctl"
            "set-volume"
            "@DEFAULT_AUDIO_SINK@"
            "5%+"
          ];
        };

        XF86AudioLowerVolume = {
          allow-when-locked = true;
          action.spawn = [
            "wpctl"
            "set-volume"
            "@DEFAULT_AUDIO_SINK@"
            "5%-"
          ];
        };

        XF86AudioMute = {
          allow-when-locked = true;
          action.spawn = [
            "wpctl"
            "set-mute"
            "@DEFAULT_AUDIO_SINK@"
            "toggle"
          ];
        };
      };
    };
  };
}
