{ pkgs, config, ... }:
let
  backgroundPath = builtins.toString ../../../assets/background.jpg;
in
{
  home.packages = with pkgs; [ hyprpaper ];
  # why the fuck services don't actually install hyprpaper
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

    extraConfig = builtins.readFile ./config.conf;
  };
}
