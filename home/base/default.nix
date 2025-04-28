{ pkgs, config, ... }:
{

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

    sing-box

    ani-cli
  ];

  home.stateVersion = "24.11";

  programs.home-manager.enable = true;
}
