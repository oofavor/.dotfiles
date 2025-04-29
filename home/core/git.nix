{
  # Basic configuration of git
  programs.git = {
    enable = true;
    userName = "oofavor";
    userEmail = "favorxog@gmail.com";
  };

  # Github cli
  programs.gh = {
    enable = true;
    gitCredentialHelper = {
      enable = true;
    };
  };
}
