{
  pkgs,
  ...
}:
{
  imports = [
    ./browsers
    ./clients
    ./games
    ./editors
    ./terminals
    ./productivity
  ];

  config = {
    # WARN:
    # this is a DUMP!!!
    # organize it ASAP
    home.packages = with pkgs; [
      wvkbd
    ];
    services.kdeconnect = {
      enable = true;
      package = pkgs.gnomeExtensions.gsconnect;
      indicator = true;
    };

    # cursor theme (not ide)
    home.file.".icons/default".source = "${pkgs.bibata-cursors}/share/icons/Bibata-Modern-Classic";
  };
}
