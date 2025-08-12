{ pkgs, inputs, ... }:
{
  imports = [ ./niri.nix ];

  environment.systemPackages = with pkgs; [
    grim # screenshot functionality
    slurp # screenshot functionality
    wl-clipboard # wl-copy and wl-paste for copy/paste from stdin / stdout
    cliphist
  ];

  programs.thunar.enable = true;
  programs.waybar.enable = true;
}
