{
  pkgs,
  config,
  lib,
  ...
}:
let
  colors = config.lib.stylix.colors.withHashtag;
  inherit (config.ofavor.desktop) lockCommand;
  niriMsg = "${lib.getExe config.programs.niri.package} msg";

  # Quickshell IPC, e.g. `qs "panels" "toggle" "control"`.
  qs =
    target: fn: args:
    [
      "quickshell"
      "ipc"
      "call"
      target
      fn
    ]
    ++ args;

  # Brightness goes through the shell so the OSD shows; plain brightnessctl
  # keeps the keys working if the shell isn't running.
  brightness = shellArgs: fallback: [
    "sh"
    "-c"
    "quickshell ipc call brightness ${shellArgs} || brightnessctl set ${fallback}"
  ];

  # Login layout: terminal with Throne to its right on "terminal", Zen on
  # "browser". Throne starts only once the terminal's window exists so the
  # columns come out in that order; the window rules below pick the workspace.
  startupApps = pkgs.writeShellApplication {
    name = "niri-startup-apps";
    runtimeInputs = [
      config.programs.niri.package
      pkgs.coreutils
      pkgs.jq
    ];
    text = ''
      window_id() {
        niri msg --json windows | jq -er --arg app "$1" 'first(.[] | select(.app_id == $app) | .id)'
      }

      wait_for() {
        for _ in $(seq 100); do
          window_id "$1" && return 0
          sleep 0.1
        done
        return 1
      }

      ghostty &
      wait_for com.mitchellh.ghostty >/dev/null || true

      Throne &
      zen-twilight &

      # Throne takes focus when it opens; hand it back to the terminal.
      wait_for Throne >/dev/null || true
      if term=$(window_id com.mitchellh.ghostty); then
        niri msg action focus-window --id "$term"
      fi
    '';
  };
in
{
  config = lib.mkIf config.ofavor.desktop.enable {

    home.packages = with pkgs; [
      brightnessctl
      xwayland-satellite # niri starts it on demand for X11 apps
    ];

    services.swayidle = {
      enable = true;
      timeouts = [
        {
          timeout = 150;
          command = "${pkgs.brightnessctl}/bin/brightnessctl -s set 10";
          resumeCommand = "${pkgs.brightnessctl}/bin/brightnessctl -r";
        }
        {
          timeout = 300;
          command = lockCommand;
        }
        {
          timeout = 330;
          command = "${niriMsg} action power-off-monitors";
          resumeCommand = "${niriMsg} action power-on-monitors";
        }
      ];
      events = {
        before-sleep = lockCommand;
        # Rearm the fingerprint reader for the lock screen.
        after-resume = "${lib.getExe config.programs.quickshell.package} ipc call lock wake";
        lock = lockCommand;
      };
    };

    programs.swaylock = {
      enable = true;
      settings = {
        image = "${../../../assets/background.jpg}";
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

      spawn-at-startup = [ { argv = [ (lib.getExe startupApps) ]; } ];

      prefer-no-csd = true;
      hotkey-overlay.skip-at-startup = true;
      screenshot-path = "~/Pictures/Screenshots/%Y-%m-%d_%H-%M-%S.png";

      cursor.hide-when-typing = true;

      layout = {
        gaps = 10;
        center-focused-column = "never";
        always-center-single-column = true;
        background-color = "transparent";

        default-column-width.proportion = 0.5;
        preset-column-widths = [
          { proportion = 1.0 / 3.0; }
          { proportion = 0.5; }
          { proportion = 2.0 / 3.0; }
          { proportion = 1.0; }
        ];

        # Sky-to-horizon gradient ring on the focused window, nothing on the rest.
        border.enable = false;
        focus-ring = {
          enable = true;
          width = 2;
          active.gradient = {
            from = colors.base0C;
            to = colors.base09;
            angle = 135;
            relative-to = "workspace-view";
          };
          inactive.color = "transparent";
          urgent.color = colors.base08;
        };

        shadow = {
          enable = true;
          softness = 28;
          spread = 4;
          offset = {
            x = 0;
            y = 6;
          };
          color = "#00000080";
        };

        tab-indicator = {
          hide-when-single-tab = true;
          place-within-column = true;
          width = 4;
          gap = 4;
          corner-radius = 2;
          active.color = colors.base09;
          inactive.color = colors.base03;
        };
      };

      overview = {
        zoom = 0.45;
        workspace-shadow.enable = true;
      };

      window-rules = [
        {
          geometry-corner-radius =
            let
              r = 12.0;
            in
            {
              top-left = r;
              top-right = r;
              bottom-left = r;
              bottom-right = r;
            };
          clip-to-geometry = true;
          draw-border-with-background = false;
        }
        {
          # Dialogs and small utilities float instead of taking a column.
          matches = [
            { app-id = "^pwvucontrol$"; }
            { app-id = "^com\\.saivert\\.pwvucontrol$"; }
            { app-id = "^nm-connection-editor$"; }
            { app-id = "^\\.?blueman-manager(-wrapped)?$"; }
            { app-id = "^org\\.gnome\\.Calculator$"; }
            { app-id = "^nomacs$"; }
            { title = "^Picture-in-Picture$"; }
            { app-id = "^org\\.kde\\.polkit-kde-authentication-agent-1$"; }
            { title = "^(Open|Save) (File|Folder|As)"; }
          ];
          open-floating = true;
        }
        {
          matches = [
            {
              app-id = "^com\\.mitchellh\\.ghostty$";
              at-startup = true;
            }
            {
              app-id = "^Throne$";
              at-startup = true;
            }
          ];
          open-on-workspace = "terminal";
        }
        {
          matches = [
            {
              app-id = "^zen-twilight$";
              at-startup = true;
            }
          ];
          open-on-workspace = "browser";
        }
        {
          matches = [ { is-active = false; } ];
          opacity = 0.96;
        }
      ];

      layer-rules = [
        {
          # Blurred wallpaper copy drawn by Quickshell sits behind the overview.
          matches = [ { namespace = "^qs-backdrop$"; } ];
          place-within-backdrop = true;
        }
        {
          matches = [ { namespace = "^qs-notifications$"; } ];
          block-out-from = "screencast";
        }
      ];

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
      };

      binds = with config.lib.niri.actions; {
        # ── Workspaces ──
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

        "Mod+o".action.toggle-overview = [ ];
        "Mod+WheelScrollDown" = {
          cooldown-ms = 150;
          action.focus-workspace-down = [ ];
        };
        "Mod+WheelScrollUp" = {
          cooldown-ms = 150;
          action.focus-workspace-up = [ ];
        };

        # ── Focus & movement ──
        "Mod+h".action = focus-column-left;
        "Mod+l".action = focus-column-right;
        "Mod+j".action.focus-window-or-workspace-down = [ ];
        "Mod+k".action.focus-window-or-workspace-up = [ ];
        "Mod+Shift+h".action = move-column-left;
        "Mod+Shift+l".action = move-column-right;
        "Mod+Shift+j".action.move-window-down-or-to-workspace-down = [ ];
        "Mod+Shift+k".action.move-window-up-or-to-workspace-up = [ ];

        "Mod+comma".action.consume-or-expel-window-left = [ ];
        "Mod+period".action.consume-or-expel-window-right = [ ];

        # ── Sizing ──
        "Mod+minus".action = set-column-width "-10%";
        "Mod+equal".action = set-column-width "+10%";
        "Mod+s".action = maximize-column;
        "Mod+w".action.switch-preset-column-width = [ ];
        "Mod+f".action.fullscreen-window = [ ];
        "Mod+Shift+c".action.center-column = [ ];
        "Mod+t".action.toggle-column-tabbed-display = [ ];
        "Mod+v".action = toggle-window-floating;
        "Mod+Shift+v".action.switch-focus-between-floating-and-tiling = [ ];

        # ── Apps & shell ──
        "Mod+q".action = spawn "ghostty";
        "Mod+Return".action = spawn "ghostty";
        "Mod+r".action.spawn = qs "panels" "toggle" [ "launcher" ];
        "Mod+Space".action.spawn = qs "panels" "toggle" [ "launcher" ];
        "Mod+Ctrl+v".action.spawn = qs "panels" "launcher" [ ";" ];
        "Mod+Ctrl+period".action.spawn = qs "panels" "launcher" [ ":" ];
        "Mod+a".action.spawn = qs "panels" "toggle" [ "control" ];
        "Mod+b".action.spawn = qs "panels" "toggle" [ "dashboard" ];
        "Mod+n".action.spawn = qs "panels" "toggle" [ "dashboard" ];
        "Mod+Shift+n".action.spawn = qs "notifications" "toggleDnd" [ ];
        "Mod+Escape".action.spawn = qs "panels" "toggle" [ "power" ];
        "Mod+Alt+l".action.spawn = [
          "sh"
          "-c"
          lockCommand
        ];

        "Mod+c".action = close-window;
        "Mod+Shift+m".action = quit;
        "Mod+Shift+slash".action.show-hotkey-overlay = [ ];

        # ── Screenshots ──
        "Print".action.screenshot = [ ];
        "Mod+Shift+s".action.screenshot = [ ];
        "Ctrl+Print".action.screenshot-screen = [ ];
        "Alt+Print".action.screenshot-window = [ ];

        # ── Hardware keys ──
        XF86MonBrightnessUp = {
          allow-when-locked = true;
          action.spawn = brightness "increment" "+5%";
        };

        XF86MonBrightnessDown = {
          allow-when-locked = true;
          action.spawn = brightness "decrement" "5%-";
        };

        XF86AudioRaiseVolume = {
          allow-when-locked = true;
          action.spawn = [
            "wpctl"
            "set-volume"
            "-l"
            "1.0"
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

        XF86AudioMicMute = {
          allow-when-locked = true;
          action.spawn = [
            "wpctl"
            "set-mute"
            "@DEFAULT_AUDIO_SOURCE@"
            "toggle"
          ];
        };

        XF86AudioPlay = {
          allow-when-locked = true;
          action.spawn = [
            "playerctl"
            "play-pause"
          ];
        };

        XF86AudioNext = {
          allow-when-locked = true;
          action.spawn = [
            "playerctl"
            "next"
          ];
        };

        XF86AudioPrev = {
          allow-when-locked = true;
          action.spawn = [
            "playerctl"
            "previous"
          ];
        };
      };
    };
  };
}
