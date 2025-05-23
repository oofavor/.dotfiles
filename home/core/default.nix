{ pkgs, config, ... }:
{
  imports = [
    ./dev.nix
    ./git.nix
    ./tmux.nix
    ./utils.nix
    ./neovim
    ./shell.nix
    ./helix
  ];

  home.stateVersion = "24.11";

  programs.home-manager.enable = true;
}
