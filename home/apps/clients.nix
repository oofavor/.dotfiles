{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
{
  imports = [ inputs.zen-browser.homeModules.twilight ];

  config = lib.mkIf config.ofavor.desktop.enable {
    stylix.targets.zen-browser.enable = false;

    programs.zen-browser = {
      enable = true;
      setAsDefaultBrowser = true;
    };

    home.packages = with pkgs; [
      pwvucontrol
      vesktop
      obsidian
      qbittorrent
      telegram-desktop
      brave
    ];

    programs.mpv.enable = true;

    services.kdeconnect = {
      enable = true;
      package = pkgs.gnomeExtensions.gsconnect;
      indicator = true;
    };

    programs.chromium.enable = true;
  };
}
