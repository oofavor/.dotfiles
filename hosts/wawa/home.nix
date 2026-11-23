{ ... }:
{
  imports = [
    ../../home
    ../../private/harpo.nix
    ../../private/claude.nix
  ];

  ofavor.desktop.enable = true;
  ofavor.dev.android.enable = true;
  ofavor.dev.langs = [
    "rust"
    "node"
    "go"
    "python"
  ];
}
