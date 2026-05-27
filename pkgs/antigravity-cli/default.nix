{
  lib,
  stdenv,
  fetchurl,
  autoPatchelfHook,
}:

stdenv.mkDerivation rec {
  pname = "antigravity-cli";
  version = "1.0.2";

  src = fetchurl {
    url = "https://storage.googleapis.com/antigravity-public/antigravity-cli/1.0.2-6109799369277440/linux-x64/cli_linux_x64.tar.gz";
    hash = "sha512-Ex9fODBAgpNvgeyP2pqjkRIxCQ9ao7J+rVfD3l2VwO+VsoGmwC2By4K+uEmEVQBP27YvDwknPVyEu7XnoPMwhg==";
  };

  nativeBuildInputs = [ autoPatchelfHook ];

  buildInputs = [ stdenv.cc.cc.lib ];

  sourceRoot = ".";

  installPhase = ''
    runHook preInstall
    mkdir -p $out/bin
    cp antigravity $out/bin/agy
    runHook postInstall
  '';

  meta = with lib; {
    description = "Antigravity CLI";
    homepage = "https://antigravity.google";
    platforms = platforms.linux;
  };
}