{ pkgs, inputs, ... }:
{
  home.packages = with pkgs; [
    tmux-sessionizer
    gitmux
  ];

  home.file.".gitmux.conf".text = ''
    tmux:
      symbols:
          branch: "" 
          hashprefix: ':'
          ahead: ↑
          behind: ↓
          staged: '● '
          conflict: '✖ '
          modified: '✚ '
          untracked: '… '
          stashed: '⚑ '
          clean: ✔
          insertions: Σ
          deletions: Δ
      styles:
          clear: '#[fg=default]'
          state: '#[fg=red,bold]'
          branch: '#[fg=white,bold]'
          remote: '#[fg=cyan]'
          divergence: '#[fg=default]'
          staged: '#[fg=green,bold]'
          conflict: '#[fg=red,bold]'
          modified: '#[fg=red,bold]'
          untracked: '#[fg=magenta,bold]'
          stashed: '#[fg=cyan,bold]'
          clean: '#[fg=green,bold]'
          insertions: '#[fg=green]'
          deletions: '#[fg=red]'
      layout: [branch, remote-branch, divergence, flags]
      options:
          branch_max_len: 0
          branch_trim: right
          ellipsis: …
          hide_clean: false
          swap_divergence: false
          divergence_space: false
  '';

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
    clock24 = false;

    # Stop tmux+escape craziness.
    escapeTime = 0;

    # Force tmux to use /tmp for sockets (WSL2 compat)
    secureSocket = false;

    plugins = with pkgs.tmuxPlugins; [
      sensible
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
      set -g renumber-windows on

      set -g default-terminal "xterm-256color"
      set -ga terminal-overrides ",*256col*:Tc"
      set -ga terminal-overrides '*:Ss=\E[%p1%d q:Se=\E[ q'
      set-environment -g COLORTERM "truecolor"

      bind c new-window -c "#{pane_current_path}"
      bind C-o display-popup -E "tms switch"
      bind C-w command-prompt -p "Rename active session to: " "run-shell 'tms rename %1'"

      set -g status-left-length 30  # Ensure no trimming of session name
      set -g status-left "#[fg=colour0,bg=colour4,bold] #S#[fg=colour4,bg=colour0]#[fg=default,bg=default] "

      set -g window-status-format "#[fg=colour7,bg=colour0] #I: #W #[fg=default,bg=default]"
      set -g window-status-current-format "#[fg=colour2,bg=colour0]#[fg=colour0,bg=colour2,bold]#I: #W#[fg=colour2,bg=colour0]#[fg=default,bg=default]"

      set -g status-right '#(gitmux -cfg $HOME/.gitmux.conf "#{pane_current_path}")'

      set -g status-bg colour0
      set -g status-fg colour7
    '';
  };
}
