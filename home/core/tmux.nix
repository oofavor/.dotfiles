{ pkgs, inputs, ... }:
let
  tms-projects = pkgs.writeShellApplication {
    name = "tms-projects";
    runtimeInputs = with pkgs; [
      coreutils
      findutils
      jq
      tmux-sessionizer
    ];
    text = ''
      config_file="/tmp/tms-projects-$(id -u).toml"
      tmp_file="$config_file.$$"

      {
        cat <<'TOML'
display_full_path = false
search_submodules = true
recursive_submodules = false
switch_filter_unknown = false
session_sort_order = "Alphabetical"
vcs_providers = ["git"]

bookmarks = [
TOML

        if [ -d "$HOME/projects" ]; then
          find "$HOME/projects" -mindepth 1 -maxdepth 1 -type d -print \
            | sort \
            | while IFS= read -r project; do
                printf '  %s,\n' "$(printf '%s' "$project" | jq -R .)"
              done
        fi

        printf ']\n'
        cat <<'TOML'

[[search_dirs]]
path = "~/obsidian"
depth = 10

[[search_dirs]]
path = "~/.dotfiles"
depth = 10

[[search_dirs]]
path = "~/projects"
depth = 10
TOML
      } > "$tmp_file"

      mv "$tmp_file" "$config_file"
      TMS_CONFIG_FILE="$config_file" exec tms "$@"
    '';
  };
in
{
  home.packages = with pkgs; [
    tmux-sessionizer
    gitmux
  ] ++ [
    tms-projects
  ];

  home.file.".gitmux.conf".text = ''
    tmux:
      symbols:
          branch: ""
          hashprefix: ':'
          ahead: '^'
          behind: 'v'
          staged: '+'
          conflict: '!'
          modified: '*'
          untracked: '?'
          stashed: '$'
          clean: ""
          insertions: '+'
          deletions: '-'
      styles:
          clear: '#[fg=#a9b1d6]'
          state: '#[fg=#f7768e]'
          branch: '#[fg=#a9b1d6]'
          remote: '#[fg=#787c99]'
          divergence: '#[fg=#787c99]'
          staged: '#[fg=#9ece6a]'
          conflict: '#[fg=#f7768e]'
          modified: '#[fg=#e0af68]'
          untracked: '#[fg=#787c99]'
          stashed: '#[fg=#787c99]'
          clean: '#[fg=#787c99]'
          insertions: '#[fg=#9ece6a]'
          deletions: '#[fg=#f7768e]'
      layout: [branch, remote-branch, divergence, flags]
      options:
          branch_max_len: 0
          branch_trim: right
          ellipsis: …
          hide_clean: true
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
      bind C-o display-popup -E "tms-projects"
      bind C-w command-prompt -p "Rename active session to: " "run-shell 'tms rename %1'"

      set -g status on
      set -g status-position bottom
      set -g status-interval 10
      set -g status-justify left
      set -g status-style "fg=#a9b1d6,bg=default"

      set -g status-left-length 28
      set -g status-left "#[fg=#a9b1d6]#S #[fg=#3b4261]"

      set -g window-status-separator ""
      set -g window-status-format "#[fg=#565f89] #I:#W "
      set -g window-status-current-format "#[fg=#c0caf5] #I:#W "

      set -g status-right-length 80
      set -g status-right "#[fg=#565f89]#(gitmux -cfg $HOME/.gitmux.conf \"#{pane_current_path}\") #[fg=#787c99]%H:%M "

      set -g pane-border-style "fg=#1a1b26"
      set -g pane-active-border-style "fg=#3b4261"
      set -g display-panes-colour "#565f89"
      set -g display-panes-active-colour "#7aa2f7"

      set -g message-style "fg=#c0caf5,bg=#16161e"
      set -g message-command-style "fg=#c0caf5,bg=#16161e"
      set -g mode-style "fg=#c0caf5,bg=#24283b"
      set -g clock-mode-colour "#7aa2f7"
    '';
  };
}
