{ inputs, username, ... }:
{
  nix.settings = {
    trusted-users = [ username ];
    experimental-features = [
      "nix-command"
      "flakes"
    ];
    auto-optimise-store = true;
    # GC keeps the build inputs of live direnv shells, so they don't rebuild after `nh clean`.
    keep-outputs = true;
    keep-derivations = true;
  };

  # nh wraps nixos-rebuild with a package diff and replaces nix.gc for cleanup.
  programs.nh = {
    enable = true;
    flake = "/home/${username}/.dotfiles";
    clean = {
      enable = true;
      dates = "weekly";
      extraArgs = "--keep 5 --keep-since 7d";
    };
  };

  # Fresher claude-code than nixpkgs.
  nixpkgs.overlays = [ inputs.claude-code.overlays.default ];

  # Allow unfree packages
  nixpkgs.config.allowUnfree = true;
  # androidenv refuses to build the SDK (home/dev/android.nix) without this.
  nixpkgs.config.android_sdk.accept_license = true;
}
