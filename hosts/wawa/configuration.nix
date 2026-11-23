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
    ./../../modules/hardware/rtw89.nix
    ./../../modules/hardware/fingerprint.nix
    ./../../modules/programs/greeter.nix
    ./../../modules/programs/office.nix
    ./../../private/vpn.nix
  ];

  networking.hostName = "wawa"; # Define your hostname.
  system.stateVersion = "24.11"; # Did you read the comment?
  services.ucodenix.enable = true;

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;
  boot.loader.grub.configurationLimit = 10;

  # Closing the lid suspends, then hibernates to the disk swap after an hour.
  boot.resumeDevice = "/dev/disk/by-uuid/0c10a0bf-8f14-4150-a889-e033ff0a0b6c";
  services.logind.settings.Login.HandleLidSwitch = "suspend-then-hibernate";
  systemd.sleep.settings.Sleep = {
    HibernateDelaySec = "1h";
  };
}
