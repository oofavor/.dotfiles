{ config, pkgs, ... }:

{
  home.username = "ofavor";
  home.homeDirectory = "/home/ofavor";

  # Packages that should be installed to the user profile.
  home.packages = with pkgs; [
    neofetch
    yazi
    gh
    hiddify-app
    firefox

    # archives
    zip
    unzip

    # utils
    ripgrep # recursively searches directories for a regex pattern
    eza # A modern replacement for ‘ls’
    fzf # A command-line fuzzy finder

    # networking tools
    aria2 # A lightweight multi-protocol & multi-source command-line download utility

    # misc
    file
    which
    tree
    gnused
    gnutar

    # nix related
    #
    # it provides the command `nom` works just like `nix`
    # with more details log output
    nix-output-monitor

    # productivity
    glow # markdown previewer in terminal

    btop  # replacement of htop/nmon
    iotop # io monitoring
    iftop # network monitoring

    # system tools
    lm_sensors # for `sensors` command
    pciutils # lspci
    usbutils # lsusb

    neovim
    just
  ];

  home.sessionVariables = {
    EDITOR = "nvim";
    TERMINAL = "ghostty";
  };

  # basic configuration of git, please change to your own
  programs.git = {
    enable = true;
    userName = "ofavor";
    userEmail = "favorxog@gmail.com";
  };

  # starship - an customizable prompt for any shell
  programs.starship = {
    enable = true;
    settings = {
      add_newline = false;
      aws.disabled = true;
      gcloud.disabled = true;
      line_break.disabled = true;
    };
  };

  # terminal
  programs.ghostty = {
    enable = true;
    settings = {
    };
  };

  programs.bash = {
    enable = true;
    enableCompletion = true;

    # TODO add your custom bashrc here
    bashrcExtra = ''
      export PATH="$PATH:$HOME/bin:$HOME/.local/bin:$HOME/go/bin"
    '';

    shellAliases = {
    };
  };

  home.stateVersion = "24.11";

  programs.home-manager.enable = true;
}
