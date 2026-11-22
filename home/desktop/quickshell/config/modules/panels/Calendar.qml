import QtQuick
import QtQuick.Layouts
import qs.common
import qs.services
import qs.widgets

ColumnLayout {
    id: root

    property int year: Time.now.getFullYear()
    property int month: Time.now.getMonth()

    function shift(delta) {
        const d = new Date(year, month + delta, 1);
        year = d.getFullYear();
        month = d.getMonth();
    }

    function reset() {
        year = Time.now.getFullYear();
        month = Time.now.getMonth();
    }

    // 42 cells starting on the Monday on/before the 1st.
    readonly property var cells: {
        const first = new Date(year, month, 1);
        const offset = (first.getDay() + 6) % 7;
        const out = [];
        for (let i = 0; i < 42; i++) {
            const d = new Date(year, month, 1 - offset + i);
            out.push({
                "day": d.getDate(),
                "inMonth": d.getMonth() === month,
                "today": d.toDateString() === Time.now.toDateString()
            });
        }
        return out;
    }

    spacing: 6

    RowLayout {
        Layout.fillWidth: true

        StyledText {
            Layout.fillWidth: true
            Layout.leftMargin: 6
            text: Qt.formatDate(new Date(root.year, root.month, 1), "MMMM yyyy")
            font.pixelSize: Theme.fontLg
            font.weight: Font.DemiBold
            color: Theme.textStrong

            MouseArea {
                anchors.fill: parent
                onClicked: root.reset()
            }
        }

        IconButton {
            size: 28
            icon: "chevron_left"
            onClicked: root.shift(-1)
        }

        IconButton {
            size: 28
            icon: "chevron_right"
            onClicked: root.shift(1)
        }
    }

    GridLayout {
        Layout.fillWidth: true
        columns: 7
        rowSpacing: 2
        columnSpacing: 2

        Repeater {
            model: ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"]

            StyledText {
                required property string modelData
                required property int index

                Layout.fillWidth: true
                Layout.preferredHeight: 24
                horizontalAlignment: Text.AlignHCenter
                text: modelData
                font.pixelSize: Theme.fontSm
                font.weight: Font.DemiBold
                color: index >= 5 ? Theme.accent : Theme.textDim
            }
        }

        Repeater {
            model: root.cells

            Item {
                required property var modelData

                Layout.fillWidth: true
                Layout.preferredHeight: 32

                Rectangle {
                    anchors.centerIn: parent
                    width: 30
                    height: 30
                    radius: 15
                    color: parent.modelData.today ? Theme.accent : "transparent"
                }

                StyledText {
                    anchors.centerIn: parent
                    text: parent.modelData.day
                    font.features: ({ "tnum": 1 })
                    font.weight: parent.modelData.today ? Font.Bold : Font.Medium
                    color: parent.modelData.today ? Theme.onAccent : parent.modelData.inMonth ? Theme.text : Theme.textDim
                }
            }
        }
    }

    WheelHandler {
        onWheel: event => root.shift(event.angleDelta.y > 0 ? -1 : 1)
    }
}
