{ ... }:
{
  # command-not-found that works with flakes, plus `, <cmd>` to run anything from nixpkgs once.
  programs.nix-index.enable = true;
  programs.nix-index-database.comma.enable = true;
}
