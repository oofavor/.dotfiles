{ ... }:
{
  networking.networkmanager = {
    enable = true;
    wifi.backend = "iwd";
  };
  networking.firewall.enable = true;

  hardware.enableRedistributableFirmware = true;
}
