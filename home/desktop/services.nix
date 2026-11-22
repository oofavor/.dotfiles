{ lib, config, ... }:
let
  hidden = "[Desktop Entry]\nHidden=true\n";
in
{
  config = lib.mkIf config.ofavor.desktop.enable {
    # Clipboard history, browsed from the launcher (";" prefix, Mod+Ctrl+V).
    services.cliphist.enable = true;
    # Keep copied text after the app it came from closes.
    services.wl-clip-persist.enable = true;

    # Mount plugged-in drives on their own, with a notification.
    services.udiskie = {
      enable = true;
      tray = "never";
      automount = true;
      notify = true;
    };

    # Quickshell's control center covers Wi-Fi and Bluetooth, so keep these
    # tray applets from autostarting (the packages stay for their manager windows).
    xdg.configFile."autostart/nm-applet.desktop".text = hidden;
    xdg.configFile."autostart/blueman.desktop".text = hidden;
  };
}
