{
  config,
  pkgs,
  inputs,
  ...
}:

{
  imports = [
    ./hardware-configuration.nix
    inputs.ucodenix.nixosModules.default
    ./../../modules/system.nix
    ./../../modules/shell.nix
    ./../../modules/sway.nix
    ./../../modules/hyprland.nix
    ./../../modules/amdpower.nix
    ./../../modules/fonts.nix
    ./../../modules/syncthing.nix
  ];

  services.ucodenix.enable = true;

  # Bootloader.
  boot.loader.grub.enable = true;
  boot.loader.grub.device = "nodev";
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub.efiSupport = true;

  services.displayManager.ly.enable = true;

  systemd.sleep.extraConfig = ''
    HibernateDelaySec=1h
  '';

  networking.hostName = "wawa"; # Define your hostname.

  system.stateVersion = "24.11"; # Did you read the comment?
}
