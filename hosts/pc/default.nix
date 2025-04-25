{ config, pkgs, ... }:

{
  imports = [
    ./hardware-configuration.nix
    ./../../modules/system.nix
    ./../../modules/nvidia.nix
    ./../../modules/sound.nix
    ./../../modules/fonts.nix
    ./../../modules/hypr.nix
    ./../../modules/shell.nix
  ];

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.grub.useOSProber = true;
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;

  networking.hostName = "ofavor"; # Define your hostname.

  system.stateVersion = "24.11"; # Did you read the comment?
}
