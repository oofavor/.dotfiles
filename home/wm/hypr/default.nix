{ pkgs, config, ... }:
let
  backgroundPath = builtins.toString ../../assets/background.jpg;
in
{
  imports = [ ./waybar ];

  # home.packages = with pkgs; [ ];

  # xdg.configFile."hyprland".source = config.lib.file.mkOutOfStoreSymlink ./configs/hyprland.conf;

  services.hyprpaper = {
    enable = true;
    settings = {
      preload = backgroundPath;
      wallpaper = " , " + backgroundPath;
    };
  };

  wayland.windowManager.hyprland = {
    enable = true;
    package = null;
    portalPackage = null;

    extraConfig = builtins.readFile ./configs/minimal.conf;
  };

}
