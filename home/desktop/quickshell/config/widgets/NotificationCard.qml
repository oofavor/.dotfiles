import QtQuick
import QtQuick.Effects
import QtQuick.Layouts
import Quickshell
import Quickshell.Widgets
import Quickshell.Services.Notifications
import qs.common
import qs.services

Rectangle {
    id: root

    required property var notification
    property bool popup: false
    readonly property bool hovered: hover.hovered
    readonly property bool critical: notification?.urgency === NotificationUrgency.Critical

    readonly property var actionList: {
        const out = [];
        const acts = notification?.actions ?? [];
        for (let i = 0; i < acts.length; i++)
            if (acts[i].identifier !== "default" && acts[i].text)
                out.push(acts[i]);
        return out;
    }

    readonly property string iconSource: {
        const n = notification;
        if (!n)
            return "";
        return resolveIcon(n.image) || resolveIcon(n.appIcon) || resolveIcon(DesktopEntries.heuristicLookup(n.desktopEntry || n.appName)?.icon ?? "");
    }

    // Accepts paths, URLs and theme names; returns "" when nothing resolves so
    // the card falls back to a symbol instead of Qt's missing-image checkerboard.
    function resolveIcon(name) {
        if (!name)
            return "";
        if (name.startsWith("/"))
            return "file://" + name;
        if (name.startsWith("image://icon/"))
            name = name.slice(13);
        else if (name.includes("://"))
            return name;
        return Quickshell.iconPath(name, true);
    }

    function invokeDefault() {
        const acts = notification?.actions ?? [];
        for (let i = 0; i < acts.length; i++)
            if (acts[i].identifier === "default") {
                acts[i].invoke();
                return;
            }
    }

    implicitHeight: layout.implicitHeight + 24
    radius: Theme.radiusLg
    color: popup ? Theme.toastBg : Theme.surfaceHigh
    border.width: 1
    border.color: critical ? Theme.alpha(Theme.red, 0.6) : Theme.outline

    HoverHandler {
        id: hover
    }

    RectangularShadow {
        visible: root.popup
        z: -1
        anchors.fill: parent
        radius: root.radius
        blur: 22
        offset.y: 6
        color: Theme.alpha("#000000", 0.35)
    }

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.LeftButton | Qt.MiddleButton
        cursorShape: Qt.PointingHandCursor
        onClicked: mouse => {
            if (mouse.button === Qt.LeftButton)
                root.invokeDefault();
            root.notification.dismiss();
        }
    }

    RowLayout {
        id: layout
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: 12
        spacing: 12

        Item {
            Layout.preferredWidth: 38
            Layout.preferredHeight: 38
            Layout.alignment: Qt.AlignTop

            ClippingRectangle {
                anchors.fill: parent
                radius: Theme.radiusMd
                color: root.critical ? Theme.alpha(Theme.red, 0.18) : Theme.surfaceHighest

                Icon {
                    anchors.centerIn: parent
                    visible: img.status !== Image.Ready
                    icon: root.critical ? "priority_high" : "notifications"
                    size: 20
                    fill: true
                    color: root.critical ? Theme.red : Theme.textMuted
                }

                Image {
                    id: img
                    anchors.fill: parent
                    anchors.margins: root.notification?.image && !root.notification.image.startsWith("image://icon/") ? 0 : 5
                    source: root.iconSource
                    fillMode: Image.PreserveAspectCrop
                    sourceSize.width: 76
                    sourceSize.height: 76
                    asynchronous: true
                }
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            spacing: 2

            RowLayout {
                Layout.fillWidth: true
                spacing: 6

                StyledText {
                    text: root.notification?.appName || "Notification"
                    font.pixelSize: Theme.fontXs
                    font.weight: Font.DemiBold
                    color: root.critical ? Theme.red : Theme.textMuted
                }

                StyledText {
                    Layout.fillWidth: true
                    text: "· " + Notifs.timeAgo(root.notification)
                    font.pixelSize: Theme.fontXs
                    color: Theme.textDim
                }

                Rectangle {
                    implicitWidth: 20
                    implicitHeight: 20
                    radius: 10
                    color: closeArea.containsMouse ? Theme.surfaceHighest : "transparent"
                    opacity: root.hovered ? 1 : 0

                    Behavior on opacity {
                        NumberAnimation { duration: Theme.durFast }
                    }

                    Icon {
                        anchors.centerIn: parent
                        icon: "close"
                        size: 14
                        color: Theme.textMuted
                    }

                    MouseArea {
                        id: closeArea
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: root.notification.dismiss()
                    }
                }
            }

            StyledText {
                Layout.fillWidth: true
                text: root.notification?.summary ?? ""
                font.weight: Font.DemiBold
                color: Theme.textStrong
            }

            StyledText {
                Layout.fillWidth: true
                visible: text !== ""
                text: root.notification?.body ?? ""
                textFormat: Text.StyledText
                wrapMode: Text.Wrap
                maximumLineCount: root.popup ? 3 : 4
                font.weight: Font.Normal
                color: Theme.text
                linkColor: Theme.accent
                onLinkActivated: link => Qt.openUrlExternally(link)
            }

            RowLayout {
                Layout.fillWidth: true
                Layout.topMargin: 6
                visible: root.actionList.length > 0
                spacing: 6

                Repeater {
                    model: root.actionList

                    Rectangle {
                        required property var modelData

                        Layout.fillWidth: true
                        implicitHeight: 28
                        radius: Theme.radiusSm
                        color: Theme.surfaceHighest

                        StyledText {
                            anchors.centerIn: parent
                            width: parent.width - 16
                            horizontalAlignment: Text.AlignHCenter
                            text: parent.modelData.text
                            font.pixelSize: Theme.fontSm
                        }

                        StateLayer {
                            onClicked: parent.modelData.invoke()
                        }
                    }
                }
            }
        }
    }
}
