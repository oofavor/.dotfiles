{pkgs, config,  ...}: {


  hardware.graphics.enable = true;

  services.xserver.videoDrivers = ["nvidia"];
  hardware.nvidia = {
    modesetting.enable = true;

    powerManagement.enable = false;

    powerManagement.finegrained = false;

    open = true;

    nvidiaSettings = true;

    package = config.boot.kernelPackages.nvidiaPackages.stable;
  };

  programs.hyprland = {
	enable = true;
	xwayland.enable = true;
};

	environment.sessionVariables = {
  	WLR_NO_HARDWARE_CURSORS = "1";
	NIXOS_OZONE_WL = "1";
	};
}
