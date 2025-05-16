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
  };
  services.hypridle.enable = true;
  programs.hyprlock.enable = true;

  # # enable Sway window manager
  # programs.sway = {
  #   enable = true;
  #   wrapperFeatures.gtk = true;
  #   extraOptions = [ "--unsupported-gpu" ];
  # };

  programs.thunar.enable = true;
  programs.waybar.enable = true;
}
