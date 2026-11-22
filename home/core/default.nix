{ ... }:
{
  imports = [
    ./git.nix
    ./isp-proxy.nix
    ./tmux.nix
    ./utils.nix
    ./shell.nix
    ./nix.nix
  ];

  home.stateVersion = "24.11"; # Did you read the comment?
}
