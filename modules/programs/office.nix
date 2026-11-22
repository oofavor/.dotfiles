{ pkgs, ... }:
{
  environment.systemPackages = with pkgs; [
    libreoffice-qt
    hunspell
    hunspellDicts.ru_RU
  ];
}
