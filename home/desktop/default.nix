{ pkgs, ... }:
{
  imports = [
    ./niri
    ./rofi
    ./notifications.nix
    ./vicinae.nix
  ];
}
