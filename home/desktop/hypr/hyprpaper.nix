let
  backgroundPath = builtins.toString ../../../assets/background.jpg;
in
{
  services.hyprpaper = {
    enable = true;
    settings = {
      ipc = "off";
      preload = [ backgroundPath ];
      wallpaper = [ ",${backgroundPath}" ];
    };
  };
}
