{
  pkgs,
  lib,
  config,
  ...
}:
let
  # Not in nixpkgs: the upstream AppImage, wrapped. In-app updates can't touch the
  # store; scripts/update-learnharness.sh bumps version + hash (weekly PR, `just update`).
  pname = "learnharness";
  version = "0.7.0";
  src = pkgs.fetchurl {
    url = "https://github.com/amosblomqvist/learnharness/releases/download/v${version}/LearnHarness-${version}-x86_64.AppImage";
    hash = "sha256-dyteBNOAy+inKnXb/yaa62pH9NRFJxGx2oBZqHIGdhM=";
  };
  contents = pkgs.appimageTools.extractType2 { inherit pname version src; };

  learnharness = pkgs.appimageTools.wrapType2 {
    inherit pname version src;
    extraInstallCommands = ''
      install -Dm444 ${contents}/learnharness.desktop -t $out/share/applications
      substituteInPlace $out/share/applications/learnharness.desktop \
        --replace-fail 'Exec=AppRun' 'Exec=learnharness'
      cp -r ${contents}/usr/share/icons $out/share
    '';
  };
in
{
  config = lib.mkIf config.ofavor.desktop.enable {
    home.packages = [ learnharness ];
  };
}
