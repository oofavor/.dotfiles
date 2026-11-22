{
  pkgs,
  ...
}:
{
  imports = [
    ./niri.nix
    ./stylix.nix
  ];

  environment.systemPackages = with pkgs; [
    grim # screenshot functionality
    slurp # screenshot functionality
    wl-clipboard # wl-copy and wl-paste for copy/paste from stdin / stdout
    # System-wide so polkit sees its policy: "unlock with system authentication".
    bitwarden-desktop
  ];

  # file explorer
  programs.thunar = {
    enable = true;
    plugins = with pkgs; [
      thunar-archive-plugin # right-click extract/compress (through file-roller)
      thunar-volman # react to plugged-in drives
    ];
  };
  # Trash, removable drives and network locations in Thunar; thumbnails for its grid.
  services.gvfs.enable = true;
  services.udisks2.enable = true;
  services.tumbler.enable = true;

  # Setcap wrapper so recording a monitor (KMS capture) needs no password prompt.
  programs.gpu-screen-recorder.enable = true;
}
