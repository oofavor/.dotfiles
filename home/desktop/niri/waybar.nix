{ config, lib, ... }:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    programs.waybar = {
      enable = true;
      systemd.enable = true;
      settings = [
        {
          layer = "top";
          position = "top";
          height = 28;
          spacing = 0;

          modules-left = [
            "niri/workspaces"
            "niri/window"
          ];
          modules-center = [ ];
          modules-right = [
            "pulseaudio"
            "network"
            "battery"
            "clock"
            "tray"
          ];

          "niri/workspaces" = {
            format = "{value}";
          };

          "niri/window" = {
            format = "{}";
            max-length = 80;
            separate-outputs = true;
            rewrite = {
              "" = "";
            };
          };

          "clock" = {
            format = "{:%H:%M}";
            format-alt = "{:%a %d %b}";
            tooltip-format = "<tt><small>{calendar}</small></tt>";
            calendar = {
              mode = "month";
              mode-mon-col = 3;
              weeks-pos = "right";
              on-scroll = 1;
              format = {
                months = "<span color='#7aa2f7'><b>{}</b></span>";
                days = "<span color='#c0caf5'>{}</span>";
                weeks = "<span color='#9ece6a'><b>W{}</b></span>";
                weekdays = "<span color='#f7768e'><b>{}</b></span>";
                today = "<span color='#1a1b26' background='#e0af68'><b>{}</b></span>";
              };
            };
          };

          "battery" = {
            format = "{capacity}%";
            format-charging = "{capacity}%+";
            format-plugged = "{capacity}%";
            states = {
              warning = 30;
              critical = 15;
            };
          };

          "network" = {
            format-wifi = "{essid}";
            format-ethernet = "wired";
            format-disconnected = "offline";
            max-length = 18;
            tooltip-format-wifi = "{essid}\n{signalStrength}% via {ifname}\n{ipaddr}";
            tooltip-format-ethernet = "{ifname}\n{ipaddr}/{cidr}";
            tooltip-format-disconnected = "Network disconnected";
          };

          "pulseaudio" = {
            format = "{volume}%";
            format-muted = "muted";
            on-click = "pwvucontrol";
            on-scroll-up = "wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+";
            on-scroll-down = "wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-";
          };

          "tray" = {
            spacing = 6;
            icon-size = 14;
          };
        }
      ];

      style = ''
        @define-color bg #11121a;
        @define-color bg-elevated #1a1b26;
        @define-color surface #24283b;
        @define-color text #c0caf5;
        @define-color muted #787c99;
        @define-color blue #7aa2f7;
        @define-color yellow #e0af68;
        @define-color red #f7768e;

        * {
          border: 0;
          border-radius: 8px;
          font-family: "Cascadia Code NF", "Noto Sans", sans-serif;
          font-size: 12px;
          font-weight: 500;
          min-height: 0;
        }

        window#waybar {
          background: transparent;
          color: @text;
        }

        tooltip {
          background: @bg-elevated;
          border: 1px solid alpha(@text, 0.14);
          border-radius: 6px;
          color: @text;
        }

        tooltip label {
          color: @text;
          padding: 6px;
        }

        #workspaces {
          background: alpha(@bg-elevated, 0.78);
          border: 1px solid alpha(@text, 0.07);
          margin: 4px 0 4px 8px;
          padding: 0 3px;
        }

        #workspaces button {
          background: transparent;
          color: @muted;
          margin: 2px 1px;
          padding: 0 8px;
        }

        #workspaces button:not(#niri-workspace-terminal):not(#niri-workspace-browser):not(#niri-workspace-misc) {
          display: none;
        }

        #workspaces button:hover {
          background: alpha(@surface, 0.55);
          color: @text;
        }

        #workspaces button.active {
          background: alpha(@blue, 0.16);
          color: @text;
        }

        #workspaces button.urgent {
          background: alpha(@red, 0.16);
          color: @red;
        }

        #window {
          color: @muted;
          margin: 4px 0;
          padding: 0 12px;
        }

        #clock {
          color: @text;
          padding: 0 10px;
        }

        #pulseaudio,
        #network,
        #battery,
        #tray {
          color: @muted;
          background: alpha(@bg-elevated, 0.68);
          border: 1px solid alpha(@text, 0.06);
          margin: 4px 0;
          padding: 0 8px;
        }

        #pulseaudio.muted,
        #network.disconnected,
        #battery.critical {
          color: @red;
        }

        #battery.warning {
          color: @yellow;
        }

        #tray {
          margin-right: 8px;
        }
      '';
    };
  };
}
