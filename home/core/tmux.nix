{ pkgs, config, ... }:
let
  c = config.lib.stylix.colors.withHashtag;

  # `tms` wrapper: regenerates the config on every run so each directory in
  # ~/projects shows up. Git repos come from search_dirs; anything that is
  # not a repo gets bookmarked (bookmarking repos too would duplicate them).
  tms = pkgs.writeShellApplication {
    name = "tms";
    runtimeInputs = with pkgs; [
      coreutils
      findutils
      jq
    ];
    text = ''
      config_file="''${XDG_RUNTIME_DIR:-/tmp}/tms.toml"
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
          find "$HOME/projects" -mindepth 1 -maxdepth 1 -type d ! -name '.*' -print \
            | sort \
            | while IFS= read -r project; do
                [ -e "$project/.git" ] && continue
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
      TMS_CONFIG_FILE="$config_file" exec ${pkgs.tmux-sessionizer}/bin/tms "$@"
    '';
  };
in
{
  home.packages = [
    tms
    pkgs.gitmux
  ];

  home.file.".gitmux.conf".text = ''
    tmux:
      symbols:
          branch: ' '
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
          clear: '#[fg=${c.base04}]'
          state: '#[fg=${c.base08}]'
          branch: '#[fg=${c.base0C}]'
          remote: '#[fg=${c.base03}]'
          divergence: '#[fg=${c.base04}]'
          staged: '#[fg=${c.base0B}]'
          conflict: '#[fg=${c.base08}]'
          modified: '#[fg=${c.base0A}]'
          untracked: '#[fg=${c.base04}]'
          stashed: '#[fg=${c.base04}]'
          clean: '#[fg=${c.base04}]'
          insertions: '#[fg=${c.base0B}]'
          deletions: '#[fg=${c.base08}]'
      layout: [branch, remote-branch, divergence, flags]
      options:
          branch_max_len: 0
          branch_trim: right
          ellipsis: …
          hide_clean: true
          swap_divergence: false
          divergence_space: false
  '';

  # Status bar is styled by hand below.
  stylix.targets.tmux.enable = false;

  programs.tmux = {
    enable = true;
    baseIndex = 1;
    newSession = true;
    clock24 = false;
    terminal = "tmux-256color";

    # Stop tmux+escape craziness.
    escapeTime = 0;

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

      set -as terminal-features ",xterm-ghostty:RGB,xterm-256color:RGB"
      set -ga terminal-overrides '*:Ss=\E[%p1%d q:Se=\E[ q'
      set-environment -g COLORTERM "truecolor"

      bind c new-window -c "#{pane_current_path}"
      bind C-o display-popup -E "tms"
      bind C-w command-prompt -p "Rename active session to: " "run-shell 'tms rename %1'"

      set -g status on
      set -g status-position bottom
      set -g status-interval 30
      set -g status-justify left
      set -g status-style "fg=${c.base04},bg=default"

      # Session pill on the left, windows as soft tabs, git + clock on the right.
      set -g status-left-length 40
      set -g status-left "#[fg=${c.base09},bg=default]#[fg=${c.base00},bg=${c.base09},bold] #S #[fg=${c.base09},bg=default,nobold]#[default]  "

      set -g window-status-style "default"
      set -g window-status-current-style "default"
      set -g window-status-separator " "
      set -g window-status-format "#[fg=${c.base03}]#I #[fg=${c.base04}]#W#{?window_zoomed_flag, 󰊓,}"
      set -g window-status-current-format "#[fg=${c.base0C},bold]#I #[fg=${c.base06}]#W#{?window_zoomed_flag, 󰊓,}"
      set -g window-status-activity-style "fg=${c.base0A}"
      set -g window-status-bell-style "fg=${c.base08}"

      set -g status-right-length 80
      set -g status-right "#(gitmux -cfg $HOME/.gitmux.conf \"#{pane_current_path}\")  #[fg=${c.base02},bg=default]#[fg=${c.base05},bg=${c.base02}] %H:%M #[fg=${c.base02},bg=default]"

      set -g pane-border-lines single
      set -g pane-border-style "fg=${c.base02}"
      set -g pane-active-border-style "fg=${c.base0C}"
      set -g display-panes-colour "${c.base03}"
      set -g display-panes-active-colour "${c.base09}"

      set -g message-style "fg=${c.base05},bg=${c.base01}"
      set -g message-command-style "fg=${c.base05},bg=${c.base01}"
      set -g mode-style "fg=${c.base00},bg=${c.base09}"
      set -g clock-mode-colour "${c.base09}"
      set -g popup-border-lines rounded
      set -g popup-border-style "fg=${c.base03}"
    '';
  };
}
