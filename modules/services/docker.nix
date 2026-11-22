{ ... }:
{
  # Socket-activated: the daemon starts on first use instead of at boot.
  virtualisation.docker = {
    enable = true;
    enableOnBoot = false;
  };
}
