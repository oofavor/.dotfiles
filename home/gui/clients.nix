{ pkgs, inputs, ... }:
{
  # TODO: change when official packages comes out
  home.packages = with pkgs; [
    inputs.zen-browser.packages."x86_64-linux".twilight
    pwvucontrol
    vesktop
    hiddify-app
    obsidian
    qbittorrent
    telegram-desktop
  ];

  programs.mpv = {
    enable = true;
  };

  services.kdeconnect = {
    enable = true;
    package = pkgs.gnomeExtensions.gsconnect;
    indicator = true;
  };
  programs.firefox.enable = true;
}
