{
  pkgs,
  config,
  lib,
  ...
}:
{
  home.packages =
    with pkgs;
    [
      gcc
      claude-code
      codex
    ]
    ++ lib.optionals (builtins.elem "zig" config.ofavor.dev.langs) [ zig ]
    ++ lib.optionals (builtins.elem "rust" config.ofavor.dev.langs) [
      rustc
      cargo
    ]
    ++ lib.optionals (builtins.elem "node" config.ofavor.dev.langs) [
      nodejs
      typescript
      bun
      pnpm
      biome
    ]
    ++ lib.optionals (builtins.elem "go" config.ofavor.dev.langs) [
      go
      air # live reload
      delve # debugger
      golangci-lint
      gotools # goimports, godoc, gorename, etc.
    ]
    ++ lib.optionals (builtins.elem "prolog" config.ofavor.dev.langs) [ swi-prolog ];
}
