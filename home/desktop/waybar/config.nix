{
  position = "bottom";
  modules-left = [ "hyprland/workspaces" ];
  modules-center = [ "clock" ];
  modules-right = [
    "pulseaudio"
    "network"
    "backlight"
    "battery"
    "hyprland/language"
    "tray"
  ];

  "hyprland/workspaces" = {
    format = "{name}";
    on-click = "activate";
    sort-by-number = true;
  };

  "hyprland/language" = {
    format = "{short}";
  };

  tray = {
    spacing = 1;
  };

  clock = {
    interval = 60;
    format = "{:%H:%M}";
    tooltip-format = "{:%B %d, %Y}";
  };

  backlight = {
    format = "{percent}% {icon}";
    format-icons = [
      ""
      ""
    ];
  };

  battery = {
    states = {
      good = 80;
      warning = 30;
      critical = 1;
    };
    format = "{capacity}% {icon}";
    format-charging = "{capacity}% {icon}";
    format-plugged = "{capacity}% {icon}";
    format-alt = "{time} left";
    format-icons = [
      ""
      ""
      ""
      ""
      ""
    ];
  };

  network = {
    format-wifi = "{signalStrength}% ";
    format-disconnected = "Disconnected";
    tooltip-format-wifi = "{essid} ({signalStrength}%)\n{ipaddr}";
    on-click = "nm-applet";
  };

  pulseaudio = {
    format = "{volume}% {icon}";
    format-bluetooth = "{volume}% {icon}";
    format-muted = "";
    format-icons = {
      "alsa_output.pci-0000_00_1f.3.analog-stereo" = "";
      "alsa_output.pci-0000_00_1f.3.analog-stereo-muted" = "";
      headphone = "";
      hands-free = "";
      headset = "";
      default = [
        ""
        ""
      ];
    };
    scroll-step = 1;
    on-click = "pwvucontrol";
    ignored-sinks = [ "Easy Effects Sink" ];
  };
}
