{ pkgs, config, ... }: {
  imports = [ ./nvim.nix ];

  # Packages that should be installed to the user profile.
  home.packages = with pkgs; [
    gh
    neofetch
    just

    # archives
    zip
    unzip

    # utils
    ripgrep # recursively searches directories for a regex pattern
    fzf # A command-line fuzzy finder

    gnused
    gnutar

    glow # markdown previewer in terminal

    btop # replacement of htop/nmon
    powertop
    lm_sensors # for `sensors` command
    pciutils # lspci
    usbutils # lsusb
    nixfmt
    sing-box
    wl-clipboard
  ];

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

  # Editor of choice
  programs.neovim = {
    enable = true;
    defaultEditor = true;
  };

  programs.ghostty = {
    enable = true;
    settings = { theme = "tokyonight"; };
  };
  home.stateVersion = "24.11";

  programs.home-manager.enable = true;
}
