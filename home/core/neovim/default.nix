{ config, pkgs, ... }:
let
  nvimPath = "${config.home.homeDirectory}/.dotfiles/home/core/neovim/nvim";
in
{
  xdg.configFile."nvim".source = config.lib.file.mkOutOfStoreSymlink nvimPath;

  stylix.targets.neovim.enable = false;

  programs.neovim = {
    enable = true;
    plugins = with pkgs.vimPlugins; [
      nvim-treesitter.withAllGrammars
      blink-cmp
      telescope-fzf-native-nvim
      go-nvim
    ];
    extraPackages = with pkgs; [
      # lua
      lua-language-server
      stylua

      # frontend (ts/css)
      typescript-language-server
      prettierd
      tailwindcss-language-server
      vscode-langservers-extracted

      # nix
      nixfmt-rfc-style
      nil

      # go
      gopls

      # obsidian
      markdown-oxide

      #python
      pyright
      black

    ];
    defaultEditor = true;
  };
}
