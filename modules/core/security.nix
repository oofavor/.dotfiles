{ ... }:
{
  # TTL sudo for less painful rebuilds
  security.sudo.extraConfig = ''
    Defaults        timestamp_timeout=30
  '';

  security.polkit.enable = true;
}
