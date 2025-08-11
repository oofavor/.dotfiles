{ pkgs, ... }:
{
  home.packages = with pkgs; [ code-cursor ];
  programs.vscode = {
    enable = true;
    package = pkgs.vscode.fhs;
  };
}
