{ ... }: {
  programs.waybar = {
    enable = true;
    style = builtins.readFile ./styles.css;

    settings = [{
      position = "top"; # Waybar position (top|bottom|left|right)
      height = 5; # Waybar height (to be removed for auto height)
      # Choose the order of the modules
      modules-left = [ "hyprland/workspaces" "hyprland/submap" "custom/media" ];
      # modules-center = [ "hyprland/window" ];
      modules-right = [
        "pulseaudio"
        "network"
        "backlight"
        "battery"
        "clock"
        "tray"
      ];
      "hyprland/submap" = { format = ''<span style="italic">{}</span>''; };
      idle_inhibitor = {
        format = "{icon}";
        format-icons = {
          activated = "";
          deactivated = "";
        };
      };
      tray = {
        # "icon-size"= 21;
        spacing = 10;
      };
      clock = {
        format = "{=%I=%M }";
        format-alt = "{=%Y-%m-%d}";
      };
      backlight = {
        # "device"= "acpi_video1";
        format = "{percent}% ";
        format-icons = [ "" "" ];
      };
      battery = {
        states = {
          good = 80;
          warning = 30;
          critical = 15;
        };
        format = "{capacity}% ";
        format-charging = "{capacity}%";
        format-plugged = "{capacity}%";
        format-alt = "{time} ";
        format-icons = [ "" "" "" "" "" ];
      };
      network = {
        # "interface"= "wlp2*"; # (Optional) To force the use of this interface
        format-wifi = "{essid} ({signalStrength}%) ";
        format-ethernet = "{ifname}= {ipaddr}/{cidr} ";
        format-linked = "{ifname} (No IP) ";
        format-disconnected = "Disconnected ";
        format-alt = "{ifname}= {ipaddr}/{cidr}";
      };
      pulseaudio = {
        # "scroll-step"= 1; # %; can be a float
        format = "🎧 {volume}%  {format_source}";
        format-bluetooth = "{volume}%  {format_source}";
        format-bluetooth-muted = " {format_source}";
        format-muted = " {format_source}";
        format-source = "{volume}% ";
        format-source-muted = "";
        on-click = "pavucontrol";
      };
    }];
  };
}
