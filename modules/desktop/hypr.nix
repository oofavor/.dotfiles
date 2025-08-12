{ pkgs, inputs, ... }:
{
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
}
