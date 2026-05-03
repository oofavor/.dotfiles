{
  pkgs,
  ...
}:
{
  imports = [
    ./niri.nix
    ./stylix.nix
  ];

  environment.systemPackages = with pkgs; [
    grim # screenshot functionality
    slurp # screenshot functionality
    wl-clipboard # wl-copy and wl-paste for copy/paste from stdin / stdout
    cliphist
  ];

  # file explorer
  programs.thunar.enable = true;
}
