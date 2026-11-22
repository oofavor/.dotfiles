{
  boot.kernelParams = [
    "amd_pstate=active" # AMD P-State EPP driver (best frequency scaling)
    "microcode.amd_sha_check=off"
    "amdgpu.abmlevel=0" # Adaptive backlight off for good: it visibly distorts colors
    "nmi_watchdog=0" # Disable NMI watchdog — reduces CPU wakeups
    "workqueue.power_efficient=Y" # Use power-efficient workqueues
  ];

  # Reduce how often dirty pages are flushed to disk (default 500 = 5s, 1500 = 15s)
  boot.kernel.sysctl."vm.dirty_writeback_centisecs" = 1500;
  boot.kernel.sysctl."vm.laptop_mode" = 5;

  hardware.graphics.enable = true;

  # No powertop auto-tune: it would override TLP's per-device runtime PM choices.
  powerManagement.enable = true;

  # TLP and power-profiles-daemon cannot run together — disable the latter
  services.power-profiles-daemon.enable = false;

  # Weekly SSD TRIM for performance and longevity
  services.fstrim.enable = true;

  services.upower.enable = true;

  # sched_ext scheduler that packs light work onto fewer cores. --autopower reads
  # the EPP TLP sets: powersave mode on battery, balanced on AC.
  services.scx = {
    enable = true;
    scheduler = "scx_lavd";
    extraArgs = [ "--autopower" ];
  };

  services.tlp = {
    enable = true;
    settings = {

      # ── CPU ────────────────────────────────────────────────────────
      CPU_ENERGY_PERF_POLICY_ON_AC = "balance_performance";
      CPU_ENERGY_PERF_POLICY_ON_BAT = "power";

      # In amd_pstate=active mode, the only valid governors are powersave and performance.
      # The powersave governor uses EPP (Energy Performance Preference) to manage frequency.
      CPU_SCALING_GOVERNOR_ON_AC = "powersave";
      CPU_SCALING_GOVERNOR_ON_BAT = "powersave";

      CPU_BOOST_ON_AC = 1;
      CPU_BOOST_ON_BAT = 0;

      # ── GPU ────────────────────────────────────────────────────────
      # The RADEON_ keys drive amdgpu's power_dpm_force_performance_level too.
      # Not "low" on battery: lowest clocks made scrolling stutter at 2560x1600.
      RADEON_DPM_PERF_LEVEL_ON_AC = "auto";
      RADEON_DPM_PERF_LEVEL_ON_BAT = "auto";

      # ── PCIe ASPM ──────────────────────────────────────────────────
      PCIE_ASPM_ON_AC = "default";
      PCIE_ASPM_ON_BAT = "powersupersave";

      # ── Disk ───────────────────────────────────────────────────────
      SATA_ALPM_ENABLE_ON_AC = "max_performance";
      SATA_ALPM_ENABLE_ON_BAT = "med_power_with_dipm";

      # ── Runtime Power Management ───────────────────────────────────
      RUNTIME_PM_ON_AC = "on"; # No surprises on AC
      RUNTIME_PM_ON_BAT = "auto"; # Let the kernel suspend idle devices

      # ── Audio ──────────────────────────────────────────────────────
      SOUND_POWER_SAVE_ON_AC = 0;
      SOUND_POWER_SAVE_ON_BAT = 1; # Suspend audio chip when idle
      SOUND_POWER_SAVE_CONTROLLER = "Y";

      # ── USB Autosuspend ────────────────────────────────────────────
      USB_AUTOSUSPEND = 1;
      USB_EXCLUDE_BTUSB = 1; # Don't suspend BT USB — avoids reconnect delays

      # ── WiFi (power save on battery, see rtw89.nix) ────────────────
      WIFI_PWR_ON_AC = "off";
      WIFI_PWR_ON_BAT = "on";

      # ── Radios ─────────────────────────────────────────────────────
      DEVICES_TO_DISABLE_ON_BAT_NOT_IN_USE = "bluetooth";

      # ── Misc ───────────────────────────────────────────────────────
      WOL_DISABLE = "Y";
    };
  };
}
