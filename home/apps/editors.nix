{
  pkgs,
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    home.packages = with pkgs; [
      code-cursor-fhs
      (pkgs.callPackage ../../pkgs/antigravity { })
    ];
    programs.vscode = {
      enable = true;
      package = pkgs.vscode.fhs;
    };
  };
}
