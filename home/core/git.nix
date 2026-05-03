{ pkgs, ... }:
{
  home.packages = with pkgs; [ lazygit ];
  # Basic configuration of git
  programs.git = {
    enable = true;
    settings.user.name = "oofavor";
    settings.user.email = "favorxog@gmail.com";
    signing.format = "openpgp";
  };

  programs.delta = {
    enable = true;
    enableGitIntegration = true;
    options = {
      navigate = true;
      side-by-side = true;
    };
  };

  # Github cli
  programs.gh = {
    enable = true;
    gitCredentialHelper = {
      enable = true;
    };
  };
}
