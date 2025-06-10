{ pkgs, ... }:
{
  # Install fish as default shell
  programs.fish = {
    enable = true;
    loginShellInit = ''
      if status is-login
        if uwsm check may-start
          exec uwsm start default
        end
      end
    '';
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

  environment.systemPackages = with pkgs; [
    man-pages
    man-pages-posix
  ];

  documentation.dev.enable = true;
}
