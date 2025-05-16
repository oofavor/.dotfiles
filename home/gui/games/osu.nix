{
  pkgs,
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.ofavor.games.enable {
    home.packages = with pkgs; [ osu-lazer-bin ];
  };
}
