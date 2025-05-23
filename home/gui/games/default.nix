{ lib, ... }:
{
  imports = [
    ./osu.nix
    ./prism.nix
  ];

  options = {
    ofavor.games.enable = lib.mkEnableOption "Enable Games";
  };
}
