{ pkgs, ... }:
{
  services.greetd = {
    enable = true;
    settings.default_session.command = ''
      ${pkgs.tuigreet}/bin/tuigreet \
        --time \
        --remember \
        --remember-session \
        --asterisks \
        --cmd "niri --session"
    '';
  };

  # Suppress kernel log spam over the greeter
  boot.consoleLogLevel = 3;
}
