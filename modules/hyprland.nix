{ pkgs, ... }:
{
  # Install hyprland
  environment.systemPackages = with pkgs; [
    grim # screenshot functionality
    slurp # screenshot functionality
    wl-clipboard # wl-copy and wl-paste for copy/paste from stdin / stdout
    mako # notification system developed by swaywm maintainer
    waybar
  ];
  programs.hyprland = {
    enable = true;
  };
  environment.sessionVariables.NIXOS_OZONE_WL = "1";

}
