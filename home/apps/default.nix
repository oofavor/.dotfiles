{
  pkgs,
  lib,
  config,
  ...
}:
{
  imports = [
    ./clients.nix
    ./games
    ./editors.nix
    ./terminals.nix
    ./media.nix
    ./mime.nix
    ./learnharness.nix
  ];

  config = lib.mkIf config.ofavor.desktop.enable {
    stylix.icons = {
      enable = true;
      # Folder tint matches the peach accent of the palette.
      package = pkgs.papirus-icon-theme.override { color = "paleorange"; };
      dark = "Papirus-Dark";
      light = "Papirus-Light";
    };

    home.packages = with pkgs; [
      nomacs
      file-roller # archives; also the backend of Thunar's extract/compress menu
    ];

    # LibreOffice owns registrymodifications.xcu, so patch the UI locale in place
    home.activation.libreofficeRussian = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      xcu="${config.xdg.configHome}/libreoffice/4/user/registrymodifications.xcu"
      if [ ! -f "$xcu" ]; then
        run mkdir -p "$(dirname "$xcu")"
        run tee "$xcu" > /dev/null <<'EOF'
      <?xml version="1.0" encoding="UTF-8"?>
      <oor:items xmlns:oor="http://openoffice.org/2001/registry" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
      </oor:items>
      EOF
      fi
      run ${pkgs.gnused}/bin/sed -i '/oor:path="\/org.openoffice.Office.Linguistic\/General"><prop oor:name="UILocale"/d' "$xcu"
      run ${pkgs.gnused}/bin/sed -i 's|</oor:items>|<item oor:path="/org.openoffice.Office.Linguistic/General"><prop oor:name="UILocale" oor:op="fuse"><value>ru</value></prop></item>\n</oor:items>|' "$xcu"
    '';
  };
}
