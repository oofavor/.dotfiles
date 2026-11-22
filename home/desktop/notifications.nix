{ lib, config, ... }:
{
  # Notifications are served by the Quickshell desktop shell (see ./quickshell).
  config = lib.mkIf config.ofavor.desktop.enable {
    services.poweralertd.enable = true;
    services.poweralertd.extraArgs = [ ];
  };
}
