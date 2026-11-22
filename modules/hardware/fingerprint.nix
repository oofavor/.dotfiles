{ pkgs, lib, ... }:
let
  # Community driver for the Goodix GM168SEC (USB 27c6:589a), which upstream
  # libfprint lists as unsupported. The sensor talks TLS with a per-device key
  # that only Windows can provision: it lives in /etc/goodix-gm168/psk.bin
  # (back it up, recovering it needs a Windows install again).
  #
  # The 80x64 frames hold 1-2 minutiae and libfprint's NBIS matcher scores anything
  # under 10 as 0, so images are matched with SIGFM (SIFT keypoints, OpenCV) instead:
  # libfprint-sigfm.patch adds it to the image-device core, gm168-sigfm.patch opts
  # the driver in. Used by sudo, polkit and the lock screen (see below).
  gm168 = pkgs.fetchFromGitHub {
    owner = "mrcook1e-ai";
    repo = "goodix-gm168";
    rev = "7f7e3710bd6405b6bc238cf85fecb74e61b4d0da";
    hash = "sha256-yPSXUBhCXlLkP/YpPzK78RLouOCWPN9c8E4xXpvxT0M=";
  };

  libfprint = pkgs.libfprint.overrideAttrs (old: {
    # Build wiring for libfprint 1.94.100 (the driver's own patch targets 1.94.10), from
    # github.com/yakoshiq/libfprint-goodix-gm168. Also drops 589a from the unsupported list.
    patches = (old.patches or [ ]) ++ [
      ./libfprint-gm168.patch
      # SIGFM matcher from github.com/danieledisint/libfprint-goodixtls511 (LGPL-2.1).
      ./libfprint-sigfm.patch
      # Suspending with the lock screen's verify running left the reader stuck open,
      # and unusable on the lock screen after the resume.
      ./libfprint-close-suspended.patch
    ];
    buildInputs = old.buildInputs ++ [ pkgs.opencv4 ];
    postPatch = (old.postPatch or "") + ''
      mkdir -p libfprint/drivers/goodix_gm168
      cp ${gm168}/src/*.c ${gm168}/src/*.h libfprint/drivers/goodix_gm168/
      # Upstream hard-codes its author's sensor calibration, which saturates this one
      # (every zone reads 511, finger detection never fires). This swaps in what the
      # Windows driver sends this laptop's sensor (USB capture), and retries instead
      # of submitting empty frames from stale touch events.
      patch -p1 -d libfprint/drivers/goodix_gm168 < ${./gm168-calibration.patch}
      patch -p1 -d libfprint/drivers/goodix_gm168 < ${./gm168-sigfm.patch}
      # Read the PSK hash from the sensor before each handshake, as Windows does.
      # Without it, every handshake after a shutdown fails ("bad record mac").
      patch -p1 -d libfprint/drivers/goodix_gm168 < ${./gm168-psk-hash.patch}
      # A finger already on the sensor while the reader starts (lid just opened, lock
      # screen just up) ends up in the background frames, and nothing matches until
      # fprintd exits. Fall back to the last good background, kept in GM168_STATE_DIR.
      patch -p1 -d libfprint/drivers/goodix_gm168 < ${./gm168-background.patch}
      # Full init instead of reusing the sensor session after a system suspend.
      patch -p1 -d libfprint/drivers/goodix_gm168 < ${./gm168-suspend.patch}
    '';
    # The shipped hwdb still lists 589a as unsupported; regenerate it from the drivers.
    # (Not via the sync-udev-hwdb target: that first downloads the list from the wiki.)
    postBuild = (old.postBuild or "") + ''
      ./libfprint/fprint-list-udev-hwdb > ../data/autosuspend.hwdb
    '';
  });
in
{
  # fprintd would otherwise put fingerprint into every PAM service; keep it opt-in.
  # Never login/greetd: the keyring is unlocked with the login password.
  options.security.pam.services = lib.mkOption {
    type = lib.types.attrsOf (lib.types.submodule { config.fprintAuth = lib.mkDefault false; });
  };

  config = {
    # Fingerprint, then password. In a terminal, Ctrl+C at the fingerprint prompt skips
    # to the password.
    security.pam.services.sudo.fprintAuth = true;
    security.pam.services.polkit-1.fprintAuth = true;
    # Fingerprint half of the Quickshell lock screen, run next to the password-only
    # quickshell-lock stack. No password fallback: this stack never answers prompts.
    security.pam.services.quickshell-fprint = {
      fprintAuth = true;
      unixAuth = false;
    };

    services.fprintd = {
      enable = true;
      package = pkgs.fprintd.override { inherit libfprint; };
    };

    # The driver writes a protocol log per session into $HOME/.goodix-gm168 (0755)
    # unless told otherwise; keep it root-only and gone after a reboot.
    systemd.services.fprintd.environment.GM168_LOG_DIR = "/run/goodix-gm168";
    systemd.services.fprintd.serviceConfig.ReadWritePaths = [ "/run/goodix-gm168" ];
    # Where the driver keeps its last good background frame (see postPatch). Adds to
    # the unit's own StateDirectory=fprint and is 0700 like it.
    systemd.services.fprintd.environment.GM168_STATE_DIR = "/var/lib/goodix-gm168";
    systemd.services.fprintd.serviceConfig.StateDirectory = "goodix-gm168";
    systemd.tmpfiles.rules = [
      "d /etc/goodix-gm168 0700 root root -"
      "d /run/goodix-gm168 0700 root root -"
    ];

    # The logged-in user may talk to the sensor directly (reading the sealed key without
    # root, VM passthrough). Must sort before 73-seat-late, which applies uaccess.
    services.udev.packages = [
      (pkgs.writeTextDir "lib/udev/rules.d/70-goodix-gm168.rules" ''
        SUBSYSTEM=="usb", ATTRS{idVendor}=="27c6", ATTRS{idProduct}=="589a", TAG+="uaccess"
      '')
    ];
  };
}
