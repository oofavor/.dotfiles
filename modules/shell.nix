{ pkgs, ... }:
{
  # Install fish as default shell
  programs.fish = {
    enable = true;
  };

  # Starship - an customizable prompt for any shell
  programs.starship = {
    enable = true;
    settings = {
      add_newline = false;
      aws.disabled = true;
      gcloud.disabled = true;
      line_break.disabled = true;
    };
  };

  users.defaultUserShell = pkgs.fish;

  environment.systemPackages = [
    pkgs.man-pages
    pkgs.man-pages-posix
  ];

  documentation.dev.enable = true;
}
