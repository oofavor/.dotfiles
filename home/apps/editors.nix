{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    home.packages = with pkgs; [
      code-cursor-fhs
      inputs.antigravity-nix.packages.${pkgs.system}.default
    ];
    programs.vscode = {
      enable = true;
      package = pkgs.vscode.fhs;
    };
  };
}
