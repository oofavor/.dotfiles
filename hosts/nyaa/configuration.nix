{
  pkgs,
  ...
}:

{
  imports = [
    ./hardware-configuration.nix
    ./../../modules/hardware/secureboot.nix
    ./../../modules/core
    ./../../modules/services/docker.nix
    ./../../modules/services/gnome.nix
    ./../../modules/services/keyd.nix
    ./../../modules/services/printing.nix
    ./../../modules/services/sound.nix
    ./../../modules/hardware/bluetooth.nix
    ./../../modules/programs/shell.nix
    ./../../modules/desktop
    ./../../modules/hardware/nvidia.nix
    ./../../modules/programs/steam.nix
    ./../../modules/programs/greeter.nix
  ];

  networking.hostName = "nyaa"; # Define your hostname.
  system.stateVersion = "24.11"; # Did you read the comment?
  boot.blacklistedKernelModules = [ "snd_pcsp" ];

  # Bootloader.

  programs.kdeconnect.enable = true;
}
