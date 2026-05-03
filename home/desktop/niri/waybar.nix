{ config, lib, ... }:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    programs.waybar = {
      enable = true;
      systemd.enable = false;
      settings = [
        {
          layer = "top";
          position = "top";

          modules-left = [ "niri/workspaces" ];
          modules-center = [ "clock" ];
          modules-right = [
            "network"
            "battery"
            "pulseaudio"
            "tray"
          ];

          "niri/workspaces" = {
            format = "{name}";
          };

          "clock" = {
            format = "{:%H:%M}";
            format-alt = "{:%Y-%m-%d %H:%M}";
            tooltip-format = "{:%A, %B %d %Y}";
          };

          "battery" = {
            format = "{capacity}% {icon}";
            format-charging = "{capacity}% ";
            format-icons = [
              ""
              ""
              ""
              ""
              ""
            ];
            states = {
              warning = 30;
              critical = 15;
            };
          };

          "network" = {
            format-wifi = "{essid} ({signalStrength}%) ";
            format-ethernet = "{ipaddr} ";
            format-disconnected = "disconnected ";
            tooltip-format-wifi = "{signalStrength}% via {ifname}";
          };

          "pulseaudio" = {
            format = "{volume}% {icon}";
            format-muted = " muted";
            format-icons = {
              default = [
                ""
                ""
                ""
              ];
            };
            on-click = "pavucontrol";
            on-scroll-up = "wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+";
            on-scroll-down = "wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%-";
          };

          "tray" = {
            spacing = 8;
          };
        }
      ];
    };
  };
}
