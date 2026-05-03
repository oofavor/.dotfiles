{ pkgs, lib, config, ... }:
{
  imports = [
    ./clients.nix
    ./games
    ./editors.nix
    ./terminals.nix
  ];

  config = lib.mkIf config.ofavor.desktop.enable {
    stylix.icons = {
      enable = true;
      package = pkgs.adwaita-icon-theme;
      dark = "Adwaita";
      light = "Adwaita";
    };

    home.packages = with pkgs; [ nomacs ];
  };
}
