{ pkgs, config, ... }:
let
  backgroundPath = builtins.toString ../../../assets/background.jpg;
in
{
  imports = [ ../waybar ];

  services.hyprpaper = {
    enable = true;
    settings = {
      preload = [ backgroundPath ];
      wallpaper = [ ",${backgroundPath}" ];
    };
  };

  wayland.windowManager.hyprland = {
    enable = true;
    package = null;
    portalPackage = null;
    xwayland.enable = false;
    systemd.enable = true;

    extraConfig = builtins.readFile ./config.conf;
  };
}
