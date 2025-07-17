{ inputs, ... }:
{
  # TODO: change when official packages comes out
  home.packages = [
    inputs.zen-browser.packages."x86_64-linux".twilight
  ];

  programs.firefox.enable = true;
}
