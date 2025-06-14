{ pkgs, inputs, ... }:
{
  # Install hyprland
  environment.systemPackages = with pkgs; [
    grim # screenshot functionality
    slurp # screenshot functionality
    wl-clipboard # wl-copy and wl-paste for copy/paste from stdin / stdout
    mako # notification system developed by swaywm maintainer
    cliphist
  ];

  programs.hyprland = {
    enable = true;
    xwayland.enable = true;
    withUWSM = true;

    package = inputs.hyprland.packages.${pkgs.stdenv.hostPlatform.system}.hyprland;
    # make sure to also set the portal package, so that they are in sync
    portalPackage =
      inputs.hyprland.packages.${pkgs.stdenv.hostPlatform.system}.xdg-desktop-portal-hyprland;
  };

  services.hypridle.enable = true;
  programs.hyprlock.enable = true;

  programs.thunar.enable = true;
  programs.waybar.enable = true;
}
