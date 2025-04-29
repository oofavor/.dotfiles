{
  boot.kernelParams = [
    "amd_pstate=active"
    "microcode.amd_sha_check=off"
  ];

  hardware.graphics.enable = true;

  powerManagement.enable = true;

  services.tlp = {
    enable = true;
    settings = {
      # CPU Settings
      CPU_ENERGY_PERF_POLICY_ON_AC = "balance_performance";
      CPU_ENERGY_PERF_POLICY_ON_BAT = "power";

      CPU_SCALING_GOVERNOR_ON_AC = "schedutil";
      CPU_SCALING_GOVERNOR_ON_BAT = "schedutil";
      CPU_MAX_PERF_ON_BAT = 50; # Limit max CPU frequency on battery

      # Boost Settings
      CPU_BOOST_ON_AC = 1;
      CPU_BOOST_ON_BAT = 0;
      CPU_HWP_DYN_BOOST_ON_AC = 1;
      CPU_HWP_DYN_BOOST_ON_BAT = 0;

      # Platform Profile
      PLATFORM_PROFILE_ON_AC = "balanced";
      PLATFORM_PROFILE_ON_BAT = "low-power";

      # GPU Settings
      AMDGPU_ABM_LEVEL_ON_AC = 0;
      AMDGPU_ABM_LEVEL_ON_BAT = 3;
      AMDGPU_DPM_STATE_ON_AC = "performance";
      AMDGPU_DPM_STATE_ON_BAT = "battery";

      # Power Management
      RUNTIME_PM_ON_AC = "auto";
      RUNTIME_PM_ON_BAT = "auto";
      PCIE_ASPM_ON_BAT = "powersupersave";

      # Wireless
      WIFI_PWR_ON_AC = "on";
      WIFI_PWR_ON_BAT = "on";
      WOL_DISABLE = "Y";
    };
  };

  services.upower.enable = true;
}
