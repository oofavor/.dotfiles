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
      inputs.antigravity-nix.packages.x86_64-linux.default
      inputs.antigravity-nix.packages.x86_64-linux.google-antigravity-ide
      inputs.antigravity-nix.packages.x86_64-linux.google-antigravity-cli
    ];
    programs.vscode = {
      enable = true;
      package = pkgs.vscode.fhs;
    };
  };
}
