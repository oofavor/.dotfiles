{ ... }:
{
  # TODO: move this to home-manager
  services.syncthing = {
    enable = true;
    openDefaultPorts = true;
    settings.gui = {
      user = "ofavor";
      password = "ofavor123";
    };
  };
}
