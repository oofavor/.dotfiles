{ lib, ... }:
let
  # Realtek RTL8852BE used to drop connections with power saving on older kernels.
  # Power saving is back on (big battery win); flip this if drops come back.
  disablePowerSave = false;
in
{
  config = lib.mkIf disablePowerSave {
    networking.networkmanager.connectionConfig."wifi.powersave" = 2;

    boot.extraModprobeConfig = ''
      options rtw89_core disable_ps_mode=y
      options rtw89_pci disable_clkreq=y
      options rtw89_pci disable_aspm_l1=y
      options rtw89_pci disable_aspm_l1ss=y
    '';

    services.tlp.settings = {
      WIFI_PWR_ON_BAT = lib.mkForce "off";
      RUNTIME_PM_DENYLIST = "01:00.0";
    };
  };
}
