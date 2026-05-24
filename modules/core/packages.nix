{ pkgs, ... }:
{
  environment.systemPackages = with pkgs; [
    vim
    wget
    curl
    git
    sysstat
    lm_sensors
    yazi
    bluez

    networkmanagerapplet
  ];
}
