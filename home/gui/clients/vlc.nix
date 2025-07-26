{ pkgs, ... }:
{
  home.packages = with pkgs; [
    vlc
  ];

  programs.mpv = {
    enable = true;
    config = {
      # # General settings
      # profile = "high-quality"; # Use high-quality profile
      # keep-open = "yes"; # Keep player open after playback
      # save-position-on-quit = "yes"; # Save playback position
      # cursor-autohide = 1000; # Hide cursor after 1 second
      # cache-default = 4000000; # Large cache for smooth streaming
      #
      # # Video settings
      # vo = "gpu-next"; # Modern GPU rendering
      # gpu-api = "vulkan"; # Preferred API, fallback to opengl if needed
      # hwdec = "auto-safe"; # Safe hardware decoding
      # scale = "ewa_lanczossharp"; # High-quality upscaling
      # scale-antiring = 0.6; # Reduce ringing artifacts
      # dither-depth = 8; # Match common monitor bit depth
      # video-sync = "audio"; # Sync video to audio for smooth playback
      #
      # # Audio settings
      # volume = 50; # Default volume
      # volume-max = 150; # Allow up to 150% volume
      # audio-channels = "stereo,5.1,7.1"; # Support common channel layouts
      # audio-spdif = "ac3,dts,eac3,dts-hd,truehd"; # Passthrough for advanced audio formats
      # audio-exclusive = "yes"; # Exclusive audio mode for soundbars
      # af = "loudnorm=I=-16:LRA=11:TP=-1.5"; # Audio normalization for consistent volume
      #
      # # Subtitle settings
      # slang = "en,eng"; # Prefer English subtitles
      # alang = "en,eng"; # Prefer English audio
      # subs-with-matching-audio = "forced"; # Only select forced subtitles if audio matches
      #
      # # OSD and UI settings
      # osc = "no"; # Disable default OSD, rely on uosc
      # osd-bar = "no"; # Disable volume bar
      # osd-font-size = 14; # Smaller OSD font
      # osd-align-x = "left"; # Align OSD to left
      # osd-align-y = "top"; # Align OSD to top
      #
      # # Single instance with umpv
      # script-opts = "umpv=1"; # Enable umpv for single-instance behavior
    };
  };
}
