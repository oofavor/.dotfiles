{ pkgs, ... }:
{
  # Packages that should be installed to the user profile.
  home.packages = with pkgs; [
    gh # github
    fastfetch # yes
    just # like make
    zip
    unzip
    ripgrep
    gnused # sed
    gnutar # tar
    glow # MD render
    btop # htop
    powertop
    lm_sensors # for `sensors` command
    pciutils # lspci
    usbutils # lsusb
    ani-cli
  ];
}
