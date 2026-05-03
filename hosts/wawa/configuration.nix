{ inputs, ... }:
{
  imports = [
    inputs.ucodenix.nixosModules.default
    ./hardware-configuration.nix
    ./../../modules/system.nix
    ./../../modules/shell.nix
    ./../../modules/desktop
    ./../../modules/hardware/amdpower.nix
    ./../../modules/greeter.nix
    ./../../modules/vpn.nix
  ];

  networking.hostName = "wawa"; # Define your hostname.
  system.stateVersion = "24.11"; # Did you read the comment?
  services.ucodenix.enable = true;

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;

  systemd.sleep.settings.Sleep = {
    HibernateDelaySec = "1h";
  };
}
