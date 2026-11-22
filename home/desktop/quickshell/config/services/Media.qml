pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Services.Mpris

Singleton {
    id: root

    readonly property var players: Mpris.players.values
    // Prefer whatever is playing; otherwise the most recently listed player.
    readonly property MprisPlayer player: players.find(p => p.isPlaying) ?? players[0] ?? null
    readonly property bool active: player !== null && (player.trackTitle ?? "") !== ""
    readonly property bool playing: player?.isPlaying ?? false
    readonly property string title: player?.trackTitle || "Nothing playing"
    readonly property string artist: player?.trackArtist || player?.identity || ""
    readonly property string artUrl: player?.trackArtUrl ?? ""
}
