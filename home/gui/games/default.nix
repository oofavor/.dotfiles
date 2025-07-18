{ lib, ... }:
{
  imports = [
    ./osu.nix
    ./prism.nix
    ./lutris.nix
  ];

  options = {
    ofavor.games.enable = lib.mkEnableOption "Enable Games";
  };
}
