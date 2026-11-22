{
  pkgs,
  ...
}:
{
  programs.niri.enable = true;
  programs.niri.package = pkgs.niri;

  xdg.portal = {
    enable = true;
    extraPortals = [ pkgs.xdg-desktop-portal-gtk ];
    config = {
      common.default = [ "gtk" ];
      # gnome is the only portal with ScreenCast for niri (screen sharing, OBS).
      # File pickers stay gtk: gnome's would need Nautilus.
      niri = {
        default = [
          "gnome"
          "gtk"
        ];
        "org.freedesktop.impl.portal.FileChooser" = [ "gtk" ];
        "org.freedesktop.impl.portal.Access" = [ "gtk" ];
        "org.freedesktop.impl.portal.Notification" = [ "gtk" ];
        "org.freedesktop.impl.portal.Secret" = [ "gnome-keyring" ];
      };
    };
  };

  # Stylix exports QT_STYLE_OVERRIDE=kvantum, which the KDE polkit agent's QML dialog
  # can't load: it segfaulted on every prompt, so all polkit authentication failed.
  systemd.user.services.niri-flake-polkit.serviceConfig.UnsetEnvironment = [ "QT_STYLE_OVERRIDE" ];

  # Password half of the Quickshell lock screen; swaylock stays as its fallback.
  security.pam.services.swaylock = { };
  security.pam.services.quickshell-lock = { };
}
