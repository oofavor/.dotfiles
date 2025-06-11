{ pkgs, inputs, ... }:
{
  imports = [
    ./hypridle.nix
    ./hyprpaper.nix
  ];

  services.hyprpolkitagent.enable = true;

  wayland.windowManager.hyprland = {
    enable = true;
    package = null;
    portalPackage = null;
    systemd.enable = false;

    extraConfig = builtins.readFile ./config.conf;

    plugins = [
      inputs.hyprland-plugins.packages.${pkgs.stdenv.hostPlatform.system}.hyprscrolling
    ];
  };
}
