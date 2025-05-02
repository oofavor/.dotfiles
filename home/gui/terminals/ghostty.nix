{
  programs.ghostty = {
    enable = true;
    settings = {
      theme = "tokyonight";
      font-family = "Cascadia Code NF";
      # no ligatures!!!
      font-feature = "-calt, -liga, -dlig";
    };
  };
}
