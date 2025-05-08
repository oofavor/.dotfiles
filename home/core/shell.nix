{ pkgs, ... }:
{
  home.packages = with pkgs; [
    fzf
    zoxide
    fd
    bat
  ];

  programs.fish = {
    enable = true;
    interactiveShellInit = ''
      set fish_greeting # Disable greeting
    '';
    plugins = with pkgs.fishPlugins; [
      {
        name = "fzf-fish";
        src = fzf-fish.src;
      }
    ];
  };
  programs.bash.enable = true;

  programs.fzf.enableFishIntegration = true;
  programs.fzf.enableBashIntegration = true;
  programs.zoxide.enableFishIntegration = true;
  programs.zoxide.enableBashIntegration = true;

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
