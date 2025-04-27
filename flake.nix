{
  description = "My NixOS + Home Manager config for PC, Laptop, and WSL";

  nixConfig = {
    extra-substituters = [ "https://nix-community.cachix.org" ];
    extra-trusted-public-keys = [
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
    ];
  };

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    cpu-microcodes = {
      url =
        "github:platomav/CPUMicrocodes/ec5200961ecdf78cf00e55d73902683e835edefd";
      flake = false;
    };
    ucodenix = {
      url = "github:e-tho/ucodenix";
      inputs.cpu-microcodes.follows = "cpu-microcodes";
    };
  };

  outputs = { self, nixpkgs, home-manager, ... }@inputs: {
    nixosConfigurations = {
      pc = let
        username = "ofavor";
        specialArgs = {
          inherit username;
          inherit inputs;
        };
      in nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        inherit specialArgs;
        modules = [
          ./hosts/pc
          ./hosts/pc/hardware-configuration.nix
          home-manager.nixosModules.home-manager
          {
            home-manager.useGlobalPkgs = true;
            home-manager.useUserPackages = true;
            home-manager.users.ofavor = import ./hosts/pc/home.nix;
          }
        ];
      };

      # TODO
      laptop = let
        username = "ofavor";
        specialArgs = {
          inherit username;
          inherit inputs;
        };
      in nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        inherit specialArgs;
        modules = [
          ./hosts/laptop
          ./hosts/laptop/hardware-configuration.nix
          home-manager.nixosModules.home-manager
          {
            home-manager.useGlobalPkgs = true;
            home-manager.useUserPackages = true;
            home-manager.users.ofavor = import ./hosts/laptop/home.nix;
          }
        ];
      };

      # TODO
      wsl = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          ./hosts/wsl/configuration.nix
          home-manager.nixosModules.home-manager
          {
            home-manager.useGlobalPkgs = true;
            home-manager.useUserPackages = true;
            home-manager.users.ofavor = import ./hosts/wsl/home.nix;
          }
        ];
      };
    };
  };
}
