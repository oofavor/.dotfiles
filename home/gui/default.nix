{
  pkgs,
  ...
}:
{
  imports = [
    ./clients
    ./games
    ./editors
    ./terminals
  ];

  # WARN:
  # this is a DUMP!!!
  # organize it ASAP
  stylix.iconTheme = {
    enable = true;
    package = pkgs.adwaita-icon-theme;
    dark = "Adwaita";
    light = "Adwaita";
  };

  home.packages = with pkgs; [ nomacs ];
}
