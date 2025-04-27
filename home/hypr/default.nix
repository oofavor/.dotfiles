{ pkgs, ... }: {

  imports = [ ./waybar ];

  wayland.windowManager.hyprland = {
    enable = false;
    package = null;
    portalPackage = null;

    # extraConfig = builtins.readFile ./hyprland.conf;
  };
}
