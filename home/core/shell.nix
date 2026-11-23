{ pkgs, config, ... }:
let
  c = config.lib.stylix.colors.withHashtag;
in
{
  # EDITOR comes from programs.neovim.defaultEditor, TERMINAL from apps/terminals.nix.
  home.sessionVariables = {
    SUDO_EDITOR = "vim";
    MANPAGER = "nvim -c '+Man!'";
    NIXOS_OZONE_WL = "1";
  };
  home.sessionPath = [
    "$HOME/.local/bin"
    "$HOME/.npm-global/bin"
  ];

  programs.fish = {
    enable = true;
    interactiveShellInit = ''
      set fish_greeting # Disable greeting
    '';
    plugins = with pkgs.fishPlugins; [
      {
        name = "fzf-fish";
        inherit (fzf-fish) src;
      }
    ];

    shellAliases = {
      cd = "z";
    };
  };

  programs.fzf = {
    enable = true;
    enableFishIntegration = true;
    # Ctrl-R belongs to atuin.
    historyWidget.command = "";
  };

  programs.zoxide = {
    enable = true;
    enableFishIntegration = true;
  };

  programs.fd.enable = true;
  programs.bat.enable = true;

  programs.atuin = {
    enable = true;
    enableFishIntegration = true;
    settings = {
      enter_accept = true;
      style = "compact";
    };
  };

  programs.eza = {
    enable = true;
    enableFishIntegration = true;
    icons = "auto";
    git = true;
    extraOptions = [ "--group-directories-first" ];
  };

  # Starship - an customizable prompt for any shell
  # Uses its own named palette below instead of Stylix's generic base16 one.
  stylix.targets.starship.enable = false;
  programs.starship = {
    enable = true;
    settings = {
      add_newline = false;
      aws.disabled = true;
      gcloud.disabled = true;
      line_break.disabled = true;

      format = "$directory$git_branch$git_state$git_status$nix_shell$python$nodejs$rust$golang$zig$cmd_duration$character";

      palette = "dusk";
      palettes.dusk = {
        peach = c.base09;
        teal = c.base0C;
        rose = c.base08;
        amber = c.base0A;
        sage = c.base0B;
        sky = c.base0D;
        lavender = c.base0E;
        muted = c.base04;
        dim = c.base03;
      };

      directory = {
        style = "bold peach";
        read_only = " 󰌾";
        read_only_style = "rose";
        truncation_length = 3;
        truncation_symbol = "…/";
        format = "[$path]($style)[$read_only]($read_only_style) ";
      };

      git_branch = {
        symbol = " ";
        style = "teal";
        format = "[$symbol$branch(:$remote_branch)]($style) ";
      };
      git_state.style = "amber";
      git_status = {
        style = "rose";
        format = "([$all_status$ahead_behind]($style) )";
      };

      nix_shell = {
        symbol = " ";
        style = "sky";
        format = "[$symbol$state]($style) ";
        impure_msg = "";
        pure_msg = "pure";
      };

      python = {
        symbol = " ";
        style = "muted";
        format = "[$symbol$pyenv_prefix($version)( $virtualenv)]($style) ";
      };
      nodejs = {
        symbol = " ";
        style = "muted";
        format = "[$symbol($version)]($style) ";
      };
      rust = {
        symbol = " ";
        style = "muted";
        format = "[$symbol($version)]($style) ";
      };
      golang = {
        symbol = " ";
        style = "muted";
        format = "[$symbol($version)]($style) ";
      };
      zig = {
        symbol = " ";
        style = "muted";
        format = "[$symbol($version)]($style) ";
      };

      cmd_duration = {
        min_time = 2000;
        style = "amber";
        format = "[󱎫 $duration]($style) ";
      };

      character = {
        success_symbol = "[❯](bold peach)";
        error_symbol = "[❯](bold rose)";
        vimcmd_symbol = "[❮](bold sage)";
      };
    };
  };

  programs.btop = {
    enable = true;
    settings = {
      rounded_corners = true;
      vim_keys = true;
      theme_background = false;
      update_ms = 1500;
      proc_tree = false;
      graph_symbol = "braille";
    };
  };
}
