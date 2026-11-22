{
  pkgs,
  config,
  lib,
  inputs,
  ...
}:
let
  # Upstream's stable toolchain instead of nixpkgs' rustc, which only has std for the host:
  # Android builds need it for the phone and emulator ABIs too. Also brings clippy, rustfmt
  # and a rust-analyzer that matches the compiler (neovim picks it up from PATH).
  rust = (inputs.rust-overlay.lib.mkRustBin { } pkgs).stable.latest.default.override {
    extensions = [
      "rust-src"
      "rust-analyzer"
    ];
    targets = lib.optionals config.ofavor.dev.android.enable [
      "aarch64-linux-android" # phones
      "armv7-linux-androideabi" # old 32-bit phones
      "x86_64-linux-android" # emulator
      "i686-linux-android"
    ];
  };
in
{
  # Per-project dev shells: `echo "use flake" > .envrc && direnv allow`.
  programs.direnv = {
    enable = true;
    nix-direnv.enable = true;
    config.global.hide_env_diff = true;
  };

  home.packages =
    with pkgs;
    [
      gcc
      codex
      opencode
    ]
    ++ lib.optionals (builtins.elem "zig" config.ofavor.dev.langs) [ zig ]
    ++ lib.optionals (builtins.elem "rust" config.ofavor.dev.langs) [ rust ]
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
    ++ lib.optionals (builtins.elem "prolog" config.ofavor.dev.langs) [ swi-prolog ]
    ++ lib.optionals (builtins.elem "python" config.ofavor.dev.langs) [
      (python3.withPackages (ps: [
        ps.playwright # headless screenshots (huashu-design verify/polish scripts)
        ps.pillow
      ]))
      uv
      ruff
      pyright
    ];

  # Playwright's own browser downloads don't run on NixOS; point it at the nixpkgs build.
  home.sessionVariables = lib.mkIf (builtins.elem "python" config.ofavor.dev.langs) {
    PLAYWRIGHT_BROWSERS_PATH = "${pkgs.playwright-driver.browsers}";
    PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS = "true";
  };
}
