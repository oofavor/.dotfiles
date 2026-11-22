pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Services.Pipewire

// What is being captured right now, read from live PipeWire links into app streams:
// a microphone, a camera device, or any other video source (niri's screencast).
Singleton {
    id: root

    readonly property var links: Pipewire.linkGroups.values.filter(g => g.state === PwLinkState.Active && g.source && g.target?.isStream)

    readonly property var micLinks: links.filter(g => g.source.type === PwNodeType.AudioSource)
    readonly property var videoLinks: links.filter(g => g.source.type === PwNodeType.VideoSource)
    readonly property var cameraLinks: videoLinks.filter(g => isCamera(g.source))
    readonly property var screenLinks: videoLinks.filter(g => !isCamera(g.source))

    readonly property bool mic: micLinks.length > 0
    readonly property bool camera: cameraLinks.length > 0
    readonly property bool screen: screenLinks.length > 0

    function isCamera(node) {
        return /^(v4l2|libcamera)/.test(node.name);
    }

    // "Firefox, Discord" for a set of links, from the consuming streams.
    function apps(groups) {
        const names = groups.map(g => g.target.properties["application.name"] || g.target.description || g.target.name);
        return [...new Set(names.filter(n => n))].join(", ");
    }

    // Stream properties are only populated while bound.
    PwObjectTracker {
        objects: root.links.map(g => g.target)
    }
}
