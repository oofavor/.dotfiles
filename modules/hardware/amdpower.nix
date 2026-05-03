{
  boot.kernelParams = [
    "amd_pstate=active"         # AMD P-State EPP driver (best frequency scaling)
    "microcode.amd_sha_check=off"
    "amdgpu.abmlevel=0"         # TLP overrides this per power state
    "nmi_watchdog=0"            # Disable NMI watchdog — reduces CPU wakeups
  ];

  # Reduce how often dirty pages are flushed to disk (default 500 = 5s, 1500 = 15s)
  boot.kernel.sysctl."vm.dirty_writeback_centisecs" = 1500;

  hardware.graphics.enable = true;

  powerManagement.enable = true;

  # TLP and power-profiles-daemon cannot run together — disable the latter
  services.power-profiles-daemon.enable = false;

  # Weekly SSD TRIM for performance and longevity
  services.fstrim.enable = true;

  services.upower.enable = true;

  services.tlp = {
    enable = true;
    settings = {

      # ── CPU ────────────────────────────────────────────────────────
      CPU_ENERGY_PERF_POLICY_ON_AC  = "balance_performance";
      CPU_ENERGY_PERF_POLICY_ON_BAT = "power";

      CPU_SCALING_GOVERNOR_ON_AC  = "schedutil";
      CPU_SCALING_GOVERNOR_ON_BAT = "schedutil";

      # Allow CPU to idle as deep as possible on battery
      CPU_MIN_PERF_ON_AC  = 0;
      CPU_MAX_PERF_ON_AC  = 100;
      CPU_MIN_PERF_ON_BAT = 0;
      CPU_MAX_PERF_ON_BAT = 40; # 6800H at 40% is still plenty for coding/browsing

      CPU_BOOST_ON_AC          = 1;
      CPU_BOOST_ON_BAT         = 0;
      CPU_HWP_DYN_BOOST_ON_AC  = 1;
      CPU_HWP_DYN_BOOST_ON_BAT = 0;

      # ── Platform Profile ───────────────────────────────────────────
      PLATFORM_PROFILE_ON_AC  = "balanced";
      PLATFORM_PROFILE_ON_BAT = "low-power";

      # ── GPU ────────────────────────────────────────────────────────
      AMDGPU_ABM_LEVEL_ON_AC  = 0;
      AMDGPU_ABM_LEVEL_ON_BAT = 3; # Adaptive backlight — reduces panel power draw
      AMDGPU_DPM_STATE_ON_AC  = "performance";
      AMDGPU_DPM_STATE_ON_BAT = "battery";

      # ── PCIe ASPM ──────────────────────────────────────────────────
      PCIE_ASPM_ON_AC  = "default";
      PCIE_ASPM_ON_BAT = "powersupersave";

      # ── Runtime Power Management ───────────────────────────────────
      RUNTIME_PM_ON_AC  = "on";   # No surprises on AC
      RUNTIME_PM_ON_BAT = "auto"; # Let the kernel suspend idle devices

      # ── Audio ──────────────────────────────────────────────────────
      SOUND_POWER_SAVE_ON_AC      = 0;
      SOUND_POWER_SAVE_ON_BAT     = 1; # Suspend audio chip when idle
      SOUND_POWER_SAVE_CONTROLLER = "Y";

      # ── USB Autosuspend ────────────────────────────────────────────
      USB_AUTOSUSPEND  = 1;
      USB_EXCLUDE_BTUSB = 1; # Don't suspend BT USB — avoids reconnect delays

      # ── WiFi ───────────────────────────────────────────────────────
      WIFI_PWR_ON_AC  = "off"; # Full performance on AC
      WIFI_PWR_ON_BAT = "on";  # Power saving on battery

      # ── Misc ───────────────────────────────────────────────────────
      WOL_DISABLE = "Y";

      # ── Battery charge thresholds ──────────────────────────────────
      # Keeps the battery between 20–80% which significantly extends
      # long-term battery capacity. If you need a full charge, temporarily
      # disable with: sudo tlp fullcharge
      START_CHARGE_THRESH_BAT0 = 20;
      STOP_CHARGE_THRESH_BAT0  = 80;
    };
  };
}
