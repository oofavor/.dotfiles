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
}
