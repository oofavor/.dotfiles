pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Services.Notifications

// Notification daemon: tracks history and the subset currently shown as toasts.
Singleton {
    id: root

    property bool dnd: false
    property var popups: []
    // QObjects can't take ad-hoc properties, so arrival times live here by id.
    property var receivedAt: ({})

    readonly property var history: server.trackedNotifications.values.slice().reverse()
    readonly property int count: server.trackedNotifications.values.length

    function removePopup(n) {
        popups = popups.filter(p => p !== n);
    }

    function clearAll() {
        popups = [];
        for (const n of server.trackedNotifications.values.slice())
            n.dismiss();
    }

    function timeAgo(n) {
        const s = Math.max(0, (Time.now - (receivedAt[n.id] ?? Date.now())) / 1000);
        if (s < 60)
            return "now";
        if (s < 3600)
            return Math.floor(s / 60) + "m";
        if (s < 86400)
            return Math.floor(s / 3600) + "h";
        return Math.floor(s / 86400) + "d";
    }

    NotificationServer {
        id: server

        keepOnReload: false
        persistenceSupported: true
        bodySupported: true
        bodyMarkupSupported: true
        bodyHyperlinksSupported: true
        bodyImagesSupported: false
        actionsSupported: true
        imageSupported: true

        onNotification: n => {
            n.tracked = true;
            root.receivedAt[n.id] = Date.now();
            n.closed.connect(() => root.removePopup(n));
            if (!root.dnd || n.urgency === NotificationUrgency.Critical)
                root.popups = [n, ...root.popups.filter(p => p !== n)].slice(0, 4);
        }
    }

    IpcHandler {
        target: "notifications"

        function toggleDnd(): void {
            root.dnd = !root.dnd;
        }

        function clear(): void {
            root.clearAll();
        }
    }
}
