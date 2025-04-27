{ config, pkgs, ... }: {
  xdg.configFile."nvim".source = config.lib.file.mkOutOfStoreSymlink ./nvim;

  programs = {
    neovim = {
      enable = true;
      plugins = with pkgs.vimPlugins; [
        telescope-fzf-native-nvim
        nvim-treesitter.withAllGrammars
      ];
      extraPackages = with pkgs; [ lua-language-server stylua prettierd ];
    };

  };

}
