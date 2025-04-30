{ pkgs, config, ... }:
{
  imports = [ ./vscode.nix ];

  home.packages = with pkgs; [
    firefox
    hiddify-app
    rofi-wayland
    qbittorrent
    vlc

    obsidian
  ];

  programs.ghostty = {
    enable = true;
    settings = {
      theme = "tokyonight";
      font-family = "Cascadia Code NF";
      font-feature = "-calt, -liga, -dlig";
    };
  };

  home.sessionVariables = {
    TERMINAL = "ghostty";
    EDITOR = "nvim";
    SUDO_EDITOR = "vim";
  };

  home.file.".icons/default".source = "${pkgs.bibata-cursors}/share/icons/Bibata-Modern-Classic";
}
