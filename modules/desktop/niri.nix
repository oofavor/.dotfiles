{
  pkgs,
  ...
}:
{
  programs.niri.enable = true;

  xdg.portal = {
    enable = true;
    extraPortals = [ pkgs.xdg-desktop-portal-gtk ];
    config = {
      common.default = [ "gtk" ];
      niri.default = [ "gtk" ];
    };
  };

  security.pam.services.swaylock = { };
}
