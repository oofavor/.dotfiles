{ ... }:
{
  # Basic configuration of git
  programs.git = {
    enable = true;
    settings = {
      user.name = "oofavor";
      user.email = "favorxog@gmail.com";

      init.defaultBranch = "main";
      pull.rebase = true;
      push.autoSetupRemote = true;
      rebase.autoStash = true;
      rerere.enabled = true;
      merge.conflictStyle = "zdiff3";
      diff.algorithm = "histogram";
      fetch.prune = true;
    };
    ignores = [
      ".direnv/"
      "result"
      "result-*"
      "**/.claude/settings.local.json"
    ];
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
