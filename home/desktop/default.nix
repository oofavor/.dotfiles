{ pkgs, ... }:
{
  imports = [
    ./niri
    ./rofi
    ./notifications.nix
  ];
}
