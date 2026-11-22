{ pkgs, ... }:
{
  # Packages that should be installed to the user profile.
  home.packages = with pkgs; [
    just # like make
    zip
    unzip
    ripgrep
    gnused # sed
    gnutar # tar
    pciutils # lspci
    usbutils # lsusb
    jq # JSON processing
    killall
    glow # render markdown in terminal
  ];
}
