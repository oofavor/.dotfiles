{ inputs, ... }:
{
  imports = [
    inputs.ucodenix.nixosModules.default
    ./hardware-configuration.nix
    ./../../modules/core
    ./../../modules/services/docker.nix
    ./../../modules/services/gnome.nix
    ./../../modules/services/keyd.nix
    ./../../modules/services/printing.nix
    ./../../modules/services/sound.nix
    ./../../modules/hardware/bluetooth.nix
    ./../../modules/programs/shell.nix
    ./../../modules/desktop
    ./../../modules/hardware/amdpower.nix
    ./../../modules/programs/greeter.nix
    ./../../modules/services/vpn.nix
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
