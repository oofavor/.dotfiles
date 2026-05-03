default:
    @just --list

# Rebuild and switch the current or given host
switch host=`hostname`:
    sudo nixos-rebuild switch --flake .#{{host}}

# Build and set as next boot without switching
boot host=`hostname`:
    sudo nixos-rebuild boot --flake .#{{host}}

# Update all flake inputs
update:
    nix flake update

# Evaluate both configs to catch nix errors without building
check:
    nix eval .#nixosConfigurations.nyaa.config.system.build.toplevel.outPath
    nix eval .#nixosConfigurations.wawa.config.system.build.toplevel.outPath

# Delete old nix generations and store paths
gc:
    sudo nix-collect-garbage -d
    nix-collect-garbage -d

# Show what would be built before switching
dry host=`hostname`:
    sudo nixos-rebuild dry-activate --flake .#{{host}}
