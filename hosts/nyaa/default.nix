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

  networking.hostName = "nyaa"; # Define your hostname.

  # TODO: change once official package comes out
  environment.systemPackages = [
    inputs.zen-browser.packages."x86_64-linux".twilight
  ];

  system.stateVersion = "24.11"; # Did you read the comment?

}
