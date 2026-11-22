{ pkgs, ... }:
{
  # Install fish as default shell. The prompt (starship) is configured in home-manager.
  programs.fish = {
    enable = true;
  };

  users.defaultUserShell = pkgs.fish;

  # nix-index (home-manager) provides command-not-found for flakes.
  programs.command-not-found.enable = false;

  environment.systemPackages = with pkgs; [
    man-pages
    man-pages-posix
  ];

  documentation.dev.enable = true;
}
