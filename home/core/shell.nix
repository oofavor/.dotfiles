{ pkgs, ... }:
{
  home.packages = with pkgs; [
    fzf
    zoxide
  ];
  programs.fzf.enableFishIntegration = true;
  programs.zoxide.enableFishIntegration = true;

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
}
