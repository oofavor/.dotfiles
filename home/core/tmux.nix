{ pkgs, config, ... }:
{
  programs.tmux = {
    enable = true;
    # aggressiveResize = true; -- Disabled to be iTerm-friendly
    baseIndex = 1;
    newSession = true;
    # Stop tmux+escape craziness.
    escapeTime = 0;
    # Force tmux to use /tmp for sockets (WSL2 compat)
    secureSocket = false;

    plugins = with pkgs.tmuxPlugins; [
      tokyo-night-tmux
      sensible
      continuum
      resurrect
      yank
      vim-tmux-navigator
    ];

    # TODO: make styles of windows simplier
    extraConfig = ''
      unbind C-b
      set -g prefix C-\;
      bind C-\; send-prefix
      setw -g mode-keys vi
      set-option -g mouse on
      set-option -g status-position bottom

      # easy-to-remember split pane commands
      bind | split-window -h -c "#{pane_current_path}"
      bind - split-window -v -c "#{pane_current_path}"
      bind c new-window -c "#{pane_current_path}"

      set -g default-terminal "xterm-256color"
      set -ga terminal-overrides ",*256col*:Tc"
      set -ga terminal-overrides '*:Ss=\E[%p1%d q:Se=\E[ q'
      set-environment -g COLORTERM "truecolor"

      set-option -g @continuum-restore 'on'

      set -g @tokyo-night-tmux_show_datetime 0

      set -g @tokyo-night-tmux_window_id_style none
      set -g @tokyo-night-tmux_pane_id_style none
      set -g @tokyo-night-tmux_zoom_id_style none
      set -g @tokyo-night-tmux_window_tidy_icons 1
    '';
  };
}
