{
  pkgs,
  ...
}:

{
  imports = [
    ./hardware-configuration.nix
    ./../../modules/secureboot.nix
    ./../../modules/system.nix
    ./../../modules/stylix.nix
    ./../../modules/shell.nix
    ./../../modules/desktop
    ./../../modules/nvidia.nix
    ./../../modules/steam.nix
    ./../../modules/greeter.nix
  ];

  networking.hostName = "nyaa"; # Define your hostname.
  system.stateVersion = "24.11"; # Did you read the comment?
  boot.blacklistedKernelModules = [ "snd_pcsp" ];

  # Bootloader.

  programs.kdeconnect.enable = true;
}
