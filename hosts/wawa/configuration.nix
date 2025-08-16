{ inputs, ... }:
{
  imports = [
    inputs.ucodenix.nixosModules.default
    ./hardware-configuration.nix
    ./../../modules/system.nix
    ./../../modules/stylix.nix
    ./../../modules/shell.nix
    ./../../modules/desktop
    ./../../modules/amdpower.nix
    ./../../modules/greeter.nix
  ];

  networking.hostName = "wawa"; # Define your hostname.
  system.stateVersion = "24.11"; # Did you read the comment?
  services.ucodenix.enable = true;

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;

  services.displayManager.cosmic-greeter.enable = true;

  systemd.sleep.extraConfig = ''
    HibernateDelaySec=1h
  '';
}
