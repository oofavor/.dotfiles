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
    xwayland.enable = true;
    withUWSM = true;
  };

  services.hypridle.enable = true;
  programs.hyprlock.enable = true;

  programs.thunar.enable = true;
  programs.waybar.enable = true;
}
