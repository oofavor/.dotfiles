{ pkgs, config, ... }:
{
  imports = [
    ./git.nix
    ./tmux.nix
    ./utils.nix
    ./shell.nix
  ];

  home.stateVersion = "24.11";

  programs.home-manager.enable = true;
}
