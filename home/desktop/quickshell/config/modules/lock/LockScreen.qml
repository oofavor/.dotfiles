import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.common
import qs.services
import qs.widgets

// Lock surface on every output; state and PAM live in services/Lock.qml.
WlSessionLock {
    id: root

    locked: Lock.locked
    onSecureChanged: Lock.secure = secure

    WlSessionLockSurface {
        id: surface

        color: Theme.bg

        Image {
            anchors.fill: parent
            source: Generated.backdrop
            fillMode: Image.PreserveAspectCrop
            asynchronous: true
        }

        Rectangle {
            anchors.fill: parent
            color: Theme.scrim
        }

        // Pointer motion rearms a fingerprint reader that went idle.
        MouseArea {
            anchors.fill: parent
            hoverEnabled: true
            onPositionChanged: Lock.wake()
            onClicked: input.forceActiveFocus()
        }

        Column {
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height * 0.2
            spacing: 4

            StyledText {
                anchors.horizontalCenter: parent.horizontalCenter
                text: Time.format("hh:mm")
                color: Theme.textStrong
                font.pixelSize: 96
                font.weight: Font.Light
            }

            StyledText {
                anchors.horizontalCenter: parent.horizontalCenter
                text: Time.format("dddd, d MMMM")
                color: Theme.text
                font.pixelSize: Theme.fontXl
            }
        }

        Column {
            anchors.horizontalCenter: parent.horizontalCenter
            y: parent.height * 0.58
            spacing: 14

            Rectangle {
                id: field

                width: 340
                height: 52
                radius: height / 2
                color: Theme.alpha(Theme.surface, 0.8)
                border.width: 1
                border.color: Lock.status !== "" ? Theme.red : input.activeFocus ? Theme.outlineStrong : Theme.outline

                transform: Translate {
                    id: shift
                }

                SequentialAnimation {
                    id: shake
                    loops: 2
                    NumberAnimation { target: shift; property: "x"; to: -10; duration: 50 }
                    NumberAnimation { target: shift; property: "x"; to: 10; duration: 70 }
                    NumberAnimation { target: shift; property: "x"; to: 0; duration: 50 }
                }

                Connections {
                    target: Lock
                    function onRejected() {
                        shake.restart();
                    }
                    function onPasswordChanged() {
                        if (input.text !== Lock.password)
                            input.text = Lock.password;
                    }
                }

                Icon {
                    anchors.left: parent.left
                    anchors.leftMargin: 18
                    anchors.verticalCenter: parent.verticalCenter
                    icon: Lock.checking ? "hourglass_top" : "lock"
                    size: 20
                    color: Theme.textMuted
                }

                TextInput {
                    id: input

                    anchors.fill: parent
                    anchors.leftMargin: 48
                    anchors.rightMargin: 56
                    verticalAlignment: TextInput.AlignVCenter
                    echoMode: TextInput.Password
                    passwordCharacter: "●"
                    color: Theme.textStrong
                    selectionColor: Theme.accentSoft
                    font.family: Theme.fontSans
                    font.pixelSize: Theme.fontLg
                    font.letterSpacing: 2
                    clip: true
                    focus: true
                    readOnly: Lock.checking

                    Component.onCompleted: forceActiveFocus()
                    onTextChanged: if (text !== Lock.password) {
                        Lock.password = text;
                        Lock.status = "";
                    }

                    Keys.onPressed: event => {
                        Lock.wake();
                        if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                            Lock.submit();
                            event.accepted = true;
                        } else if (event.key === Qt.Key_Escape) {
                            Lock.password = "";
                            event.accepted = true;
                        }
                    }

                    StyledText {
                        anchors.fill: parent
                        visible: input.text === ""
                        text: Lock.fingerprint === "off" ? "Password" : "Password or fingerprint"
                        color: Theme.textDim
                        font.pixelSize: Theme.fontLg
                    }
                }

                // A password typed in the wrong layout is the classic lock-screen failure.
                Rectangle {
                    anchors.right: parent.right
                    anchors.rightMargin: 12
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Niri.keyboardLayouts.length > 1
                    width: layoutText.implicitWidth + 14
                    height: 26
                    radius: 13
                    color: Niri.keyboardLayoutIdx === 0 ? Theme.surfaceHighest : Theme.accentSoft

                    StyledText {
                        id: layoutText
                        anchors.centerIn: parent
                        text: Niri.keyboardLayout.toUpperCase()
                        color: Niri.keyboardLayoutIdx === 0 ? Theme.textMuted : Theme.accent
                        font.pixelSize: Theme.fontSm
                        font.weight: Font.DemiBold
                    }
                }
            }

            Row {
                anchors.horizontalCenter: parent.horizontalCenter
                spacing: 8
                height: 22

                Icon {
                    anchors.verticalCenter: parent.verticalCenter
                    visible: Lock.fingerprint !== "off" && Lock.status === ""
                    icon: "fingerprint"
                    size: 20
                    color: ({
                            "waiting": Theme.accent,
                            "retry": Theme.red
                        })[Lock.fingerprint] ?? Theme.textMuted
                }

                StyledText {
                    anchors.verticalCenter: parent.verticalCenter
                    text: Lock.status !== "" ? Lock.status : ({
                            "waiting": "Touch the sensor",
                            "retry": "Not recognized, try again",
                            "idle": "Press any key to use fingerprint"
                        })[Lock.fingerprint] ?? ""
                    color: Lock.status !== "" || Lock.fingerprint === "retry" ? Theme.red : Theme.textMuted
                    font.pixelSize: Theme.fontMd
                }
            }
        }
    }
}
