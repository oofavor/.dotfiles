{
  pkgs,
  config,
  ...
}:
{
  imports = [
    ./browsers
    ./clients
    ./games
    ./editors
    ./terminals
  ];

  # WARN:
  # this is a DUMP!!!
  # organize it ASAP
  home.packages = with pkgs; [
  ];

  # cursor theme (not ide)
  home.file.".icons/default".source = "${pkgs.bibata-cursors}/share/icons/Bibata-Modern-Classic";
}
