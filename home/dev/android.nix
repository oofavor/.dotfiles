{
  pkgs,
  config,
  lib,
  ...
}:
let
  # Versions pinned by React Native 0.88 / Expo SDK 58 (react-native/gradle/libs.versions.toml).
  # The SDK lives read-only in the store, so Gradle can't download a missing component itself:
  # when a build asks for another platform, build-tools, NDK or CMake version, add it here.
  buildToolsVersion = "37.0.0";
  ndkVersion = "27.1.12297006";

  android = pkgs.androidenv.composeAndroidPackages {
    platformVersions = [ "37.0" ];
    buildToolsVersions = [ buildToolsVersion ];
    includeNDK = true;
    ndkVersions = [ ndkVersion ];
    cmakeVersions = [ "3.22.1" ];

    # One x86_64 image for the emulator (KVM); create a device with `avdmanager create avd`.
    includeEmulator = true;
    includeSystemImages = true;
    systemImageTypes = [ "google_apis" ];
    abiVersions = [ "x86_64" ];
  };

  sdk = "${android.androidsdk}/libexec/android-sdk";
  ndk = "${sdk}/ndk/${ndkVersion}";
in
{
  config = lib.mkIf config.ofavor.dev.android.enable {
    home.packages = [
      android.androidsdk # adb, emulator, sdkmanager, avdmanager
      pkgs.watchman # Metro's file watcher
    ]
    # Points cargo at the NDK's clang and linker: `cargo ndk -t arm64-v8a build`.
    # The Android targets themselves come with the toolchain in dev.nix.
    ++ lib.optionals (builtins.elem "rust" config.ofavor.dev.langs) [ pkgs.cargo-ndk ];

    # Sets JAVA_HOME. React Native builds with JDK 17; Gradle would otherwise try to
    # download one, and prebuilt JDKs don't run on NixOS.
    programs.java = {
      enable = true;
      package = pkgs.jdk17;
    };

    home.sessionVariables = {
      ANDROID_HOME = sdk;
      # cargo-ndk, Dioxus and build scripts read the first, Tauri the second.
      ANDROID_NDK_HOME = ndk;
      NDK_HOME = ndk;
    };

    # Projects bring their own Gradle through ./gradlew; this only writes ~/.gradle/gradle.properties.
    programs.gradle = {
      enable = true;
      package = null;
      # The Android Gradle plugin downloads a prebuilt aapt2 from Maven that can't run on NixOS.
      settings."android.aapt2FromMavenOverride" = "${sdk}/build-tools/${buildToolsVersion}/aapt2";
    };
  };
}
