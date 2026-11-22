{ ... }:
{
  # Compressed RAM swap; takes priority over the disk swap, which stays for hibernation.
  zramSwap = {
    enable = true;
    memoryPercent = 50;
  };

  # Kill the biggest memory hog in the user session before the desktop freezes.
  systemd.oomd.enableUserSlices = true;

  services.fwupd.enable = true;
}
