{ pkgs, ... }:
{
  services.displayManager.autoLogin.user = "ofavor";
  services.displayManager.autoLogin.enable = true;
}
