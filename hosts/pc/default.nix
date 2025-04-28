{ config, pkgs, ... }:

{
  imports = [
    ./hardware-configuration.nix
    ./../../modules/system.nix
    ./../../modules/shell.nix
    ./../../modules/hyprland.nix
    ./../../modules/nvidia.nix
    ./../../modules/fonts.nix
  ];

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.grub.useOSProber = true;
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;

  services.displayManager.ly.enable = true;

  networking.hostName = "ofavor"; # Define your hostname.

  system.stateVersion = "24.11"; # Did you read the comment?
}
