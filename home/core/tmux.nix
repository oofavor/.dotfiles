{ pkgs, config, ... }:
let
  tokyonight = pkgs.tmuxPlugins.mkTmuxPlugin {
    pluginName = "tokyo-night-tmux";
    rtpFilePath = "tokyo-night.tmux";
    version = "v1.6.6";
    src = pkgs.fetchFromGitHub {
      owner = "janoamaral";
      repo = "tokyo-night-tmux";
      rev = "v1.6.6";
      hash = "sha256-TOS9+eOEMInAgosB3D9KhahudW2i1ZEH+IXEc0RCpU0=";
    };
  };
in
{
  home.packages = with pkgs; [ tmux-sessionizer ];
  xdg.configFile."tms/config.toml".text = ''
    [[search_dirs]]
    path = "/home/ofavor/obsidian"
    depth = 10
    search_submodules = true
    full_path = true

    [[search_dirs]]
    path = "/home/ofavor/.dotfiles"
    depth = 10

    [[search_dirs]]
    path = "/home/ofavor/projects"
    depth = 10
    search_submodules = true
    full_path = true
  '';

  programs.tmux = {
    enable = true;
    baseIndex = 1;
    newSession = true;
    # Stop tmux+escape craziness.
    escapeTime = 0;
    # Force tmux to use /tmp for sockets (WSL2 compat)
    secureSocket = false;

    plugins = with pkgs.tmuxPlugins; [
      sensible
      {
        plugin = tokyonight;
        extraConfig = ''
          set -g @tokyo-night-tmux_window_id_style none
          set -g @tokyo-night-tmux_pane_id_style hide
          set -g @tokyo-night-tmux_zoom_id_style none

          set -g @tokyo-night-tmux_window_tidy_icons 1
          set -g @tokyo-night-tmux_show_datetime 0
          set -g @tokyo-night-tmux_transparent 1
        '';
      }
      {
        plugin = resurrect;
        extraConfig = ''
          set -g @resurrect-strategy-nvim 'session'
        '';
      }
      {
        plugin = continuum;
        extraConfig = ''
          set-option -g @continuum-restore 'on'
        '';
      }
      yank
      vim-tmux-navigator
    ];

    extraConfig = ''
      unbind C-b
      set -g prefix C-\;
      bind C-\; send-prefix
      setw -g mode-keys vi
      set-option -g mouse on
      set-option -g status-position bottom
      set -g renumber-windows on
      set -g default-command "fish"

      bind c new-window -c "#{pane_current_path}"

      set -g default-terminal "xterm-256color"
      set -ga terminal-overrides ",*256col*:Tc"
      set -ga terminal-overrides '*:Ss=\E[%p1%d q:Se=\E[ q'
      set-environment -g COLORTERM "truecolor"

      bind C-o display-popup -E "tms switch"
      bind C-w command-prompt -p "Rename active session to: " "run-shell 'tms rename %1'"
    '';
  };
}
