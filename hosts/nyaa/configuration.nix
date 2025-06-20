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
    ./../../modules/secureboot.nix
    ./../../modules/shell.nix
    ./../../modules/desktop.nix
    ./../../modules/nvidia.nix
    ./../../modules/fonts.nix
    ./../../modules/steam.nix
    ./../../modules/greeter.nix
  ];

  networking.hostName = "nyaa"; # Define your hostname.
  system.stateVersion = "24.11"; # Did you read the comment?
  boot.kernelPackages = pkgs.linuxPackages_latest;
  boot.blacklistedKernelModules = [ "snd_pcsp" ];

  # Bootloader.

  programs.kdeconnect.enable = true;
}
