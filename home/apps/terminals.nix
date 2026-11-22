{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.ofavor.desktop.enable {
    home.sessionVariables = {
      TERMINAL = "ghostty";
    };

    programs.ghostty = {
      enable = true;
      # Colors, font and opacity come from Stylix.
      settings = {
        # no ligatures!!!
        font-feature = "-calt, -liga, -dlig";
        gtk-single-instance = true;
        linux-cgroup = "never";

        # Fill the whole window: no padding; leftover pixels use the plain background.
        window-padding-y = 0;
        window-padding-color = "background";
        # niri blurs the wallpaper behind the translucent background.
        background-blur = true;

        # Solid block everywhere; stop shell integration switching it to a bar at the prompt.
        cursor-style = "block";
        cursor-style-blink = false;
        shell-integration-features = "no-cursor";
        mouse-hide-while-typing = true;
      };
    };
  };
}
