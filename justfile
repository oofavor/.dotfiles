default:
    @just --list

# Rebuild and switch the current or given host (shows a package diff first)
switch host=`hostname`:
    nh os switch . -H {{host}}

# Build and set as next boot without switching
boot host=`hostname`:
    nh os boot . -H {{host}}

# Show what switching would do without doing it
dry host=`hostname`:
    nh os switch . -H {{host}} --dry

# Update all flake inputs and packages pinned outside the lock (scripts/update-*.sh)
update:
    nix flake update
    scripts/update-learnharness.sh

# Evaluate the config to catch nix errors without building
check:
    nix eval .#nixosConfigurations.wawa.config.system.build.toplevel.outPath

# Format all nix files
fmt:
    nix fmt

# Lint nix files for antipatterns and dead code
lint:
    nix run nixpkgs#statix -- check .
    nix run nixpkgs#deadnix -- --fail .

# Delete old generations and store paths (keeps the last 5 and anything from the past week)
gc:
    nh clean all --keep 5 --keep-since 7d

# Measure battery draw, wakeups and interrupts (run on battery, mostly idle)
power seconds="30":
    python3 scripts/powerstat.py {{seconds}}
