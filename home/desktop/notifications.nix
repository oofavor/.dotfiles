{ lib, config, ... }:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    services.dunst.enable = true;
    services.poweralertd.enable = true;
    services.poweralertd.extraArgs = [ ];
  };
}
