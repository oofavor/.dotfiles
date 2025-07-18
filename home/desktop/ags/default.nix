{ inputs, pkgs, ... }:
let
  astral = inputs.astal.packages.${pkgs.system};
in
{
  # add the home manager module
  imports = [ inputs.ags.homeManagerModules.default ];

  programs.ags = {
    enable = true;

    # symlink to ~/.config/ags
    configDir = ./ags;

    # additional packages and executables to add to gjs's runtime
    extraPackages = [
      astral.battery
      astral.io
    ];
  };
}
