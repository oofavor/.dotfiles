{ pkgs, lib, config, ... }:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    programs.rofi = {
      enable = true;
      plugins = with pkgs; [
        rofi-calc
      ];
    };
  };
}
