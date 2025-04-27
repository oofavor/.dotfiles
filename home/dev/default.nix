{pkgs, ...}: {
imports = [./nvim];

  home.packages = with pkgs; [zig rustc cargo];

}
