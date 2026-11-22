{ pkgs, ... }:
let
  claudeWrapped = pkgs.writeShellScriptBin "claude" ''
    set -euo pipefail
    export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
    export DISABLE_TELEMETRY=1
    export DISABLE_ERROR_REPORTING=1
    export DO_NOT_TRACK=1
    export CLAUDE_CODE_AUTO_CONNECT_IDE=0
    exec isp-run ${pkgs.claude-code}/bin/claude "$@"
  '';
in
{
  home.packages = [ claudeWrapped ];
}
