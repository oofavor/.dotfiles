{
  lib,
  buildGoModule,
  makeWrapper,
  systemd,
  libnotify,
  xray,
}:
buildGoModule {
  pname = "harpo";
  version = "2.0.0";
  src = ./.;
  vendorHash = "sha256-KzY8vi0s81hcYdo80RezQLERWeMR990uMp+Tut4mjC0=";
  subPackages = [ "cmd/harpo" ];
  env.CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
  ];
  nativeBuildInputs = [ makeWrapper ];
  postInstall = ''
    wrapProgram $out/bin/harpo \
      --set-default HARPO_XRAY "${xray}/bin/xray" \
      --prefix PATH : "${
        lib.makeBinPath [
          systemd
          libnotify
          xray
        ]
      }"
    makeWrapper $out/bin/.harpo-wrapped $out/bin/harpo-run \
      --argv0 harpo-run \
      --set-default HARPO_XRAY "${xray}/bin/xray" \
      --prefix PATH : "${
        lib.makeBinPath [
          systemd
          libnotify
          xray
        ]
      }"
  '';
}
