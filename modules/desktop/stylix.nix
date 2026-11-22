{ pkgs, ... }:
{
  stylix.enable = true;
  stylix.polarity = "dark";

  # "Dusk" — hand-tuned from assets/background.jpg: deep navy night, teal sky,
  # peach/rose horizon glow and amber city lights.
  stylix.base16Scheme = {
    scheme = "Dusk";
    author = "ofavor";
    base00 = "0c1019"; # background
    base01 = "121823"; # panels, status bars
    base02 = "1b2331"; # selection
    base03 = "364256"; # comments, borders
    base04 = "7b879b"; # muted foreground
    base05 = "cad3df"; # foreground
    base06 = "dfe6ee";
    base07 = "f2f5f9";
    base08 = "e27d84"; # rose
    base09 = "eda587"; # peach — primary accent
    base0A = "e8c38c"; # amber
    base0B = "9fcba0"; # sage
    base0C = "78c2c6"; # teal — secondary accent
    base0D = "7ea6d8"; # dusk blue
    base0E = "b49fdb"; # lavender
    base0F = "c48c7c"; # clay
  };

  stylix.cursor = {
    name = "Bibata-Modern-Classic";
    package = pkgs.bibata-cursors;
    size = 22;
  };

  stylix.fonts = {
    serif = {
      package = pkgs.noto-fonts;
      name = "Noto Serif";
    };

    sansSerif = {
      package = pkgs.inter;
      name = "Inter";
    };

    monospace = {
      package = pkgs.cascadia-code;
      name = "Cascadia Code NF";
    };

    emoji = {
      package = pkgs.noto-fonts-color-emoji;
      name = "Noto Color Emoji";
    };

    sizes = {
      terminal = 16;
      applications = 11;
      desktop = 11;
      popups = 11;
    };
  };

  stylix.opacity = {
    terminal = 0.9;
    popups = 0.97;
  };

  # Icon font used by the Quickshell desktop shell.
  fonts.packages = [ pkgs.material-symbols ];
}
