{ pkgs, ... }: {
  services.displayManager.ly.enable = true;

  programs.hyprland = { enable = true; };

  environment.sessionVariables.NIXOS_OZONE_WL = "1";

  xdg.portal.enable = true;
  xdg.portal.extraPortals = with pkgs; [ xdg-desktop-portal-gtk ];
}
