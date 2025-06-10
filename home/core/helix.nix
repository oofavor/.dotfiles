{ pkgs, lib, ... }:
{
  programs.helix = {
    enable = true;
    package = pkgs.evil-helix;
    settings = {
      theme = "autumn_night_transparent";
      editor.cursor-shape = {
        normal = "block";
        insert = "bar";
        select = "underline";
      };
    };
    languages.language = [
      {
        name = "nix";
        auto-format = true;
        formatter.command = lib.getExe pkgs.nixfmt-rfc-style;
        language-servers = [ "nil-ls" ];
      }
      {
        name = "lua";
        scope = "source.lua";
        file-types = [ "lua" ];
        auto-format = true;
        formatter.command = lib.getExe pkgs.stylua;
        language-servers = [ "lua-ls" ];
      }
    ];
    themes = {
      autumn_night_transparent = {
        "inherits" = "autumn_night";
        "ui.background" = { };
      };
    };
  };

}
