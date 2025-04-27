{ pkgs, config, ... }: {

  imports = [ ./vscode.nix ];

  home.packages = with pkgs; [
  microsoft-edge
    hiddify-app
    ghostty
    rofi-wayland
    waybar
    mako
  ];

  home.sessionVariables = { TERMINAL = "ghostty"; };
  home.file.".icons/default".source =
    "${pkgs.bibata-cursors}/share/icons/Bibata-Modern-Classic";

}
