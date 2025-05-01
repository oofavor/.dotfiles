{
  pkgs,
  config,
  ...
}:
{
  imports = [
    ./vscode.nix
  ];

  home.packages = with pkgs; [
    firefox
    hiddify-app
    rofi-wayland
    qbittorrent
    vlc

    obsidian

    # TODO: move prism to games/
    (prismlauncher.override {
      # Add binary required by some mod
      additionalPrograms = [ ffmpeg ];

      # Change Java runtimes available to Prism Launcher
      jdks = [
        graalvm-ce
        zulu8
        zulu17
        zulu
      ];
    })
  ];

  # TODO: move to programs specific files
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
    NIXOS_OZONE_WL = "1";
  };

  home.file.".icons/default".source = "${pkgs.bibata-cursors}/share/icons/Bibata-Modern-Classic";
}
