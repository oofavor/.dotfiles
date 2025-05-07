{ pkgs, config, ... }:
{
  imports = [
    ./dev.nix
    ./git.nix
    ./tmux.nix
    ./utils.nix
    ./neovim
    ./helix
  ];

  home.stateVersion = "24.11";

  home.sessionVariables = {
    EDITOR = "nvim";
    SUDO_EDITOR = "vim";
  };

  programs.home-manager.enable = true;
}
