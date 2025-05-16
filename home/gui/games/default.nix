{
  lib,
  config,
  ...
}:
with lib;
{
  options = {
    games.enable = mkEnableOption "Games";
  };

  config = mkIf config.games.enable {
    self.imports = [
      ./prism.nix
      ./osu.nix
    ];
  };
}
