pragma Singleton

import QtQuick
import Quickshell

// Design tokens. Raw palette, fonts and wallpaper come from Generated.qml,
// which Nix writes from the Stylix scheme so the shell always matches the system.
Singleton {
    id: root

    function alpha(c, a) {
        return Qt.rgba(c.r, c.g, c.b, a);
    }

    function mix(a, b, t) {
        return Qt.rgba(a.r * (1 - t) + b.r * t, a.g * (1 - t) + b.g * t, a.b * (1 - t) + b.b * t, 1);
    }

    // ── Palette ────────────────────────────────────────────────────────
    readonly property color bg: Generated.base00
    readonly property color surface: Generated.base01
    readonly property color surfaceHigh: Generated.base02
    readonly property color surfaceHighest: mix(Generated.base02, Generated.base03, 0.45)
    readonly property color outline: alpha(Generated.base05, 0.08)
    readonly property color outlineStrong: alpha(Generated.base05, 0.16)

    readonly property color text: Generated.base05
    readonly property color textStrong: Generated.base07
    readonly property color textMuted: Generated.base04
    readonly property color textDim: Generated.base03

    readonly property color accent: Generated.base09
    readonly property color accentSoft: alpha(Generated.base09, 0.16)
    readonly property color onAccent: Generated.base00
    readonly property color secondary: Generated.base0C
    readonly property color secondarySoft: alpha(Generated.base0C, 0.16)

    readonly property color red: Generated.base08
    readonly property color orange: Generated.base09
    readonly property color yellow: Generated.base0A
    readonly property color green: Generated.base0B
    readonly property color cyan: Generated.base0C
    readonly property color blue: Generated.base0D
    readonly property color purple: Generated.base0E

    readonly property color islandBg: alpha(Generated.base01, 0.55)
    readonly property color panelBg: alpha(Generated.base01, 0.72)
    readonly property color toastBg: alpha(Generated.base01, 0.97)
    readonly property color scrim: alpha("#000000", 0.35)
    readonly property color shadow: alpha("#000000", 0.45)

    // ── Typography ─────────────────────────────────────────────────────
    readonly property string fontSans: Generated.fontSans
    readonly property string fontMono: Generated.fontMono
    readonly property string fontIcon: "Material Symbols Rounded"

    readonly property int fontXs: 10
    readonly property int fontSm: 11
    readonly property int fontMd: 12
    readonly property int fontLg: 14
    readonly property int fontXl: 18
    readonly property int fontXxl: 28
    readonly property int fontHuge: 44

    // ── Geometry ───────────────────────────────────────────────────────
    readonly property int radiusSm: 8
    readonly property int radiusMd: 12
    readonly property int radiusLg: 16
    readonly property int radiusXl: 22

    readonly property int gap: 8
    readonly property int pad: 14

    readonly property int barHeight: 30
    readonly property int barMargin: 6
    readonly property int barSide: 10
    readonly property int barTotal: barHeight + barMargin

    // ── Motion ─────────────────────────────────────────────────────────
    readonly property int durFast: 140
    readonly property int durMed: 240
    readonly property int durSlow: 380

    // Material 3 "emphasized decelerate" — snappy start, long gentle settle.
    readonly property list<real> curveDecel: [0.05, 0.7, 0.1, 1, 1, 1]
    readonly property list<real> curveStandard: [0.2, 0, 0, 1, 1, 1]
    readonly property list<real> curveAccel: [0.3, 0, 0.8, 0.15, 1, 1]
}
