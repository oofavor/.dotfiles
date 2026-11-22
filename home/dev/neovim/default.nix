{
  pkgs,
  lib,
  config,
  ...
}:
let
  c = config.lib.stylix.colors.withHashtag;
  slots = map (i: "base0${lib.toHexString i}") (lib.range 0 15);
  hasLang = lang: builtins.elem lang config.ofavor.dev.langs;
in
{
  xdg.configFile."nvim".source = ./nvim;

  # The Lua config themes tokyonight with this palette (see plugins/theme.lua).
  stylix.targets.neovim.enable = false;
  xdg.dataFile."nvim/dusk-palette.lua".text = ''
    return {
    ${lib.concatMapStrings (n: "  ${n} = '${c.${n}}',\n") slots}}
  '';

  # Plugins are managed by lazy.nvim (it resets the packpath, so Nix-installed
  # plugins would never load). Nix only supplies the tools they call out to.
  programs.neovim = {
    enable = true;
    withPython3 = false;
    withRuby = false;
    extraPackages =
      with pkgs;
      [
        # lazy builds telescope-fzf-native and LuaSnip's jsregexp with make
        gnumake

        # lua
        lua-language-server
        stylua

        # frontend (ts/css)
        typescript-language-server
        prettierd
        tailwindcss-language-server
        vscode-langservers-extracted

        # nix
        nixfmt
        nil

        # go
        gopls

        # obsidian
        markdown-oxide

        #python
        pyright
        ruff

        # c/c++ (leetcode.nvim defaults to cpp)
        clang-tools
      ]
      ++ lib.optionals (hasLang "zig") [ zls ];
    defaultEditor = true;
  };

  # Opens leetcode.nvim (see nvim/lua/ofavor/plugins/leetcode.lua).
  programs.fish.shellAliases.leet = "nvim leetcode.nvim";
}
