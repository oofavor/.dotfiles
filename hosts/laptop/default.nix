{ config, pkgs, inputs, ... }:

{
  imports = [
    ./hardware-configuration.nix
    ./../../modules/system.nix
    ./../../modules/amdpower.nix
    ./../../modules/hypr.nix
    ./../../modules/sound.nix
    ./../../modules/fonts.nix
    ./../../modules/shell.nix
    inputs.ucodenix.nixosModules.default
  ];

  services.ucodenix.enable = true;

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;

  services.displayManager.ly.enable = true;

  networking.hostName = "ofavor"; # Define your hostname.

  system.stateVersion = "24.11"; # Did you read the comment?
}
