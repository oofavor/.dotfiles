{
  description = "I like descriptive descriptions";

  nixConfig = {
    extra-substituters = [
      "https://nix-community.cachix.org"
      "https://cache.nixos.org"
      "https://hyprland.cachix.org"
    ];
    extra-trusted-public-keys = [
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "hyprland.cachix.org-1:a7pgxzMz7+chwVL3/pzj6jIBMioiJM7ypFP8PwtkuGc="
    ];
  };

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    cpu-microcodes = {
      url = "github:platomav/CPUMicrocodes/ec5200961ecdf78cf00e55d73902683e835edefd";
      flake = false;
    };

    ucodenix = {
      url = "github:e-tho/ucodenix";
      inputs.cpu-microcodes.follows = "cpu-microcodes";
    };

    zen-browser = {
      url = "github:0xc000022070/zen-browser-flake";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    lanzaboote = {
      url = "github:nix-community/lanzaboote/v0.4.2";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    hyprland.url = "github:hyprwm/Hyprland";
    hyprland-plugins = {
      url = "github:hyprwm/hyprland-plugins";
      inputs.hyprland.follows = "hyprland";
    };
  };

  # TODO: Improve whatever this is please
  outputs =
    {
      self,
      nixpkgs,
      home-manager,
      ...
    }@inputs:
    {
      nixosConfigurations =
        let
          username = "ofavor";
          specialArgs = {
            inherit username;
            inherit inputs;
          };
        in
        {
          # pc
          nyaa = nixpkgs.lib.nixosSystem {
            system = "x86_64-linux";
            inherit specialArgs;
            modules = [
              inputs.lanzaboote.nixosModules.lanzaboote
              ./hosts/nyaa/configuration.nix
              ./hosts/nyaa/hardware-configuration.nix
              home-manager.nixosModules.home-manager
              {
                home-manager.useGlobalPkgs = true;
                home-manager.useUserPackages = true;
                home-manager.users.ofavor = import ./hosts/nyaa/home.nix;
                home-manager.extraSpecialArgs = specialArgs;
              }
            ];
          };

          # Xiaomi laptop
          wawa = nixpkgs.lib.nixosSystem {
            system = "x86_64-linux";
            inherit specialArgs;
            modules = [
              ./hosts/wawa/configuration.nix
              ./hosts/wawa/hardware-configuration.nix
              home-manager.nixosModules.home-manager
              {
                home-manager.useGlobalPkgs = true;
                home-manager.useUserPackages = true;
                home-manager.users.ofavor = import ./hosts/wawa/home.nix;
                home-manager.extraSpecialArgs = specialArgs;
              }
            ];
          };
        };
    };
}
