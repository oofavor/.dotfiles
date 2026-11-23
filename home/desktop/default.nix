{ ... }:
{
  imports = [
    ./niri
    ./notifications.nix
    ./quickshell
    ./services.nix
    ./screen-tools.nix
    ./lock.nix
  ];

  # rofi isn't installed, but stylix's target still sets the renamed programs.rofi.font option.
  stylix.targets.rofi.enable = false;

  # wayprompt was removed from nixpkgs/home-manager, but stylix's target still sets programs.wayprompt.
  stylix.targets.wayprompt.enable = false;
}
