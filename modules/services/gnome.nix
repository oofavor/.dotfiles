{ pkgs, ... }:
{
  programs.dconf.enable = true;
  services.gnome.gnome-keyring.enable = true;

  services = {
    dbus.packages = with pkgs; [ gcr ];
    geoclue2.enable = true;
    udev.packages = with pkgs; [ gnome-settings-daemon ];
  };
}
