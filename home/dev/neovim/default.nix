{ config, pkgs, ... }:
{
  xdg.configFile."nvim".source = config.lib.file.mkOutOfStoreSymlink ./nvim;

  programs.neovim = {
    enable = true;
    plugins = with pkgs.vimPlugins; [
      nvim-treesitter.withAllGrammars
      blink-cmp
    ];
    extraPackages = with pkgs; [
      #lua
      lua-language-server
      stylua

      # js/ts
      typescript-language-server
      prettierd
      tailwindcss-language-server
      vscode-langservers-extracted

      #nix
      nixfmt-rfc-style
      nil

    ];
  };

  home.sessionVariables = {
    EDITOR = "neovim";
  };
}
