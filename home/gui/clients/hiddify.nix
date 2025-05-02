{ pkgs, ... }:
{
  home.packages = with pkgs; [
    hiddify-app
  ];
}
