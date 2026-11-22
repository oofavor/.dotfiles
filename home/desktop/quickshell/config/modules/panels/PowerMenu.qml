import QtQuick
import QtQuick.Layouts
import Quickshell
import qs.common
import qs.services
import qs.widgets

// Session menu: arrow keys / hover to choose, Enter or click to run.
Item {
    id: root

    property bool open: false
    property int selected: 0

    readonly property var actions: [
        {
            "icon": "lock",
            "label": "Lock",
            "run": () => Quickshell.execDetached(Generated.lockCommand)
        },
        {
            "icon": "bedtime",
            "label": "Suspend",
            "run": () => Quickshell.execDetached(["systemctl", "suspend"])
        },
        {
            "icon": "logout",
            "label": "Log out",
            "run": () => Niri.quit()
        },
        {
            "icon": "restart_alt",
            "label": "Restart",
            "run": () => Quickshell.execDetached(["systemctl", "reboot"])
        },
        {
            "icon": "power_settings_new",
            "label": "Shut down",
            "run": () => Quickshell.execDetached(["systemctl", "poweroff"])
        }
    ]

    function activate(i) {
        Panels.close();
        actions[i].run();
    }

    implicitWidth: content.implicitWidth
    implicitHeight: content.implicitHeight
    visible: opacity > 0
    opacity: open ? 1 : 0
    scale: open ? 1 : 0.94
    focus: open

    onOpenChanged: if (open) {
        selected = 0;
        forceActiveFocus();
    }

    Behavior on opacity {
        NumberAnimation { duration: Theme.durMed }
    }

    Behavior on scale {
        NumberAnimation { duration: Theme.durMed; easing.type: Easing.BezierSpline; easing.bezierCurve: Theme.curveDecel }
    }

    Keys.onLeftPressed: selected = (selected + actions.length - 1) % actions.length
    Keys.onRightPressed: selected = (selected + 1) % actions.length
    Keys.onTabPressed: selected = (selected + 1) % actions.length
    Keys.onReturnPressed: activate(selected)
    Keys.onEnterPressed: activate(selected)
    Keys.onSpacePressed: activate(selected)

    ColumnLayout {
        id: content
        anchors.centerIn: parent
        spacing: 22

        StyledText {
            Layout.alignment: Qt.AlignHCenter
            text: "See you soon, " + (Quickshell.env("USER") || "friend")
            font.pixelSize: Theme.fontXl
            font.weight: Font.DemiBold
            color: Theme.textStrong
        }

        Row {
            Layout.alignment: Qt.AlignHCenter
            spacing: 14

            Repeater {
                model: root.actions

                Rectangle {
                    id: tile

                    required property var modelData
                    required property int index
                    readonly property bool isSelected: root.selected === index

                    width: 104
                    height: 112
                    radius: Theme.radiusXl
                    color: isSelected ? Theme.accent : Theme.panelBg
                    border.width: 1
                    border.color: isSelected ? "transparent" : Theme.outline

                    Behavior on color {
                        ColorAnimation { duration: Theme.durFast }
                    }

                    ColumnLayout {
                        anchors.centerIn: parent
                        spacing: 10

                        Icon {
                            Layout.alignment: Qt.AlignHCenter
                            icon: tile.modelData.icon
                            size: 32
                            fill: tile.isSelected
                            color: tile.isSelected ? Theme.onAccent : Theme.text
                        }

                        StyledText {
                            Layout.alignment: Qt.AlignHCenter
                            text: tile.modelData.label
                            color: tile.isSelected ? Theme.onAccent : Theme.textMuted
                        }
                    }

                    MouseArea {
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onEntered: root.selected = tile.index
                        onClicked: root.activate(tile.index)
                    }
                }
            }
        }

        StyledText {
            Layout.alignment: Qt.AlignHCenter
            text: "←  →  to choose  ·  Enter to confirm  ·  Esc to cancel"
            font.pixelSize: Theme.fontSm
            color: Theme.alpha(Theme.text, 0.5)
        }
    }
}
