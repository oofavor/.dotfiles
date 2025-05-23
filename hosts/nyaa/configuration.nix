{
  config,
  pkgs,
  inputs,
  ...
}:

{
  imports = [
    ./hardware-configuration.nix
    ./../../modules/system.nix
    ./../../modules/shell.nix
    ./../../modules/desktop.nix
    ./../../modules/nvidia.nix
    ./../../modules/fonts.nix
  ];

  networking.hostName = "nyaa"; # Define your hostname.
  system.stateVersion = "24.11"; # Did you read the comment?
  boot.kernelPackages = pkgs.linuxPackages_latest;
  boot.blacklistedKernelModules = [ "snd_pcsp" ];

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;
  boot.loader.grub.useOSProber = true;

  services.displayManager.ly.enable = true;

  programs.kdeconnect.enable = true;
}
