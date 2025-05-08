{ pkgs, ... }:
{
  # Packages that should be installed to the user profile.
  home.packages = with pkgs; [
    gh
    neofetch
    just
    zip
    unzip
    ripgrep
    gnused
    gnutar
    glow
    btop
    powertop
    lm_sensors # for `sensors` command
    pciutils # lspci
    usbutils # lsusb
    sing-box
    ani-cli
  ];
}
