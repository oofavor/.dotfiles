{ pkgs, ... }:
{
  home.packages = with pkgs; [
    zig
    rustc
    cargo
    nodejs
    typescript
    go
    gcc

    nix-prefetch-github
  ];
}
