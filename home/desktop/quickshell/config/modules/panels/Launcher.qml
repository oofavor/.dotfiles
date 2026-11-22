import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import Quickshell.Widgets
import qs.common
import qs.services
import qs.widgets

// App launcher. Plain text searches apps; a leading character switches mode:
//   =  calculate with qalc (automatic for input that starts like an expression)
//   >  run a shell command in the terminal, Shift+Enter runs it in the background
//   +  append a line to the Obsidian inbox note
//   :  emoji picker, Enter copies
//   ;  clipboard history, Enter copies, Shift+Enter deletes the entry
Popout {
    id: root

    readonly property int rowHeight: 48
    readonly property int maxRows: 8

    property int selected: 0

    readonly property string query: input.text
    readonly property string mode: ({
            "=": "calc",
            ">": "run",
            "+": "note",
            ":": "emoji",
            ";": "clip"
        })[query.charAt(0)] ?? "apps"
    readonly property string arg: (mode === "apps" ? query : query.slice(1)).trim()

    // Starts with a digit/bracket and has an operator or space after it, so
    // "2^10" and "5 km to mi" calculate but "7zip" stays an app search.
    readonly property bool looksLikeMath: /^[-(.\d]/.test(arg) && /[-+*\/^%()\s]/.test(arg.slice(1))
    readonly property bool calcActive: mode === "calc" || (mode === "apps" && looksLikeMath)
    property string calcFor: ""
    property string calcResult: ""
    readonly property bool calcReady: calcActive && arg !== "" && calcFor === arg && calcResult !== ""

    readonly property string noteName: Generated.notesFile.split("/").pop()

    // "😀\tgrinning face" per line, generated from Unicode's emoji-test.txt.
    property var emojis: []
    readonly property var emojiResults: {
        if (mode !== "emoji")
            return [];
        const words = arg.toLowerCase().split(/\s+/).filter(w => w !== "");
        return emojis.filter(e => words.every(w => e.name.includes(w))).slice(0, 60);
    }

    // `cliphist list` lines: "<id>\t<preview>", newest first.
    property var clipEntries: []
    readonly property var clipResults: {
        if (mode !== "clip")
            return [];
        const q = arg.toLowerCase();
        return clipEntries.filter(l => l.toLowerCase().includes(q)).slice(0, 100);
    }

    readonly property var apps: DesktopEntries.applications.values.filter(a => !a.noDisplay)

    readonly property var appResults: {
        if (mode !== "apps")
            return [];
        const q = arg.toLowerCase();
        const counts = usage.counts;
        return apps.map(app => ({
                    app,
                    score: q === "" ? counts[app.id] ?? 0 : score(app, q, counts)
                })).filter(r => r.score >= 0).sort((a, b) => b.score - a.score || a.app.name.localeCompare(b.app.name)).map(r => r.app);
    }

    readonly property var items: {
        const out = [];
        if (calcReady)
            out.push({
                kind: "calc",
                icon: "calculate",
                title: calcResult,
                subtitle: arg + "  ·  Enter to copy"
            });
        else if (mode === "calc")
            out.push({
                kind: "hint",
                icon: "calculate",
                title: arg === "" ? "Type an expression" : "…",
                subtitle: "Units, currencies, percentages, functions"
            });
        if (mode === "run")
            out.push({
                kind: "run",
                icon: "terminal",
                title: arg || "Type a command",
                subtitle: "Enter: run in terminal  ·  Shift+Enter: run in background"
            });
        if (mode === "note")
            out.push({
                kind: "note",
                icon: "edit_note",
                title: arg || "Type a note",
                subtitle: "Enter: add to " + noteName
            });
        if (mode === "emoji" && emojiResults.length === 0)
            out.push({
                kind: "hint",
                icon: "mood",
                title: emojis.length === 0 ? "Loading…" : "No matching emoji",
                subtitle: "Search by name: heart, thumbs up, flag"
            });
        for (const e of emojiResults)
            out.push({
                kind: "emoji",
                emoji: e.char,
                title: e.name
            });
        if (mode === "clip" && clipResults.length === 0)
            out.push({
                kind: "hint",
                icon: "content_paste",
                title: clipEntries.length === 0 ? "Clipboard history is empty" : "No matching entries"
            });
        for (const line of clipResults) {
            const preview = line.slice(line.indexOf("\t") + 1);
            out.push({
                kind: "clip",
                line,
                icon: preview.startsWith("[[ binary data") ? "image" : "content_paste",
                title: preview,
                subtitle: "Enter: copy  ·  Shift+Enter: delete"
            });
        }
        for (const app of appResults)
            out.push({
                kind: "app",
                app,
                title: app.name,
                subtitle: app.genericName || app.comment
            });
        return out;
    }

    function score(app, q, counts) {
        const name = app.name.toLowerCase();
        let s;
        if (name === q)
            s = 100;
        else if (name.startsWith(q))
            s = 80;
        else if (name.split(/[\s\-_.]+/).some(w => w.startsWith(q)))
            s = 60;
        else if (name.includes(q))
            s = 40;
        else if ([app.genericName, app.id, ...app.keywords].some(t => t && t.toLowerCase().includes(q)))
            s = 25;
        else if (isSubsequence(q, name))
            s = 10;
        else
            return -1;
        // Frequently launched apps float up, capped so a strong text match still wins.
        return s + Math.min(18, 4 * Math.log2(1 + (counts[app.id] ?? 0)));
    }

    function isSubsequence(q, s) {
        let i = 0;
        for (const c of s)
            if (c === q[i] && ++i === q.length)
                return true;
        return false;
    }

    function inTerminal(command) {
        return Generated.terminal.concat(["-e"], command);
    }

    function activate(i, shift) {
        const item = items[i];
        if (!item)
            return;
        switch (item.kind) {
        case "app":
            {
                const counts = Object.assign({}, usage.counts);
                counts[item.app.id] = (counts[item.app.id] ?? 0) + 1;
                usage.counts = counts;
                Niri.spawn(item.app.runInTerminal ? inTerminal(item.app.command) : item.app.command);
                break;
            }
        case "calc":
            Niri.spawn([Generated.wlCopy, item.title]);
            break;
        case "run":
            if (!arg)
                return;
            Niri.spawn(shift ? ["sh", "-c", arg] : inTerminal(["sh", "-c", arg + '; exec "${SHELL:-sh}"']));
            break;
        case "note":
            if (!arg)
                return;
            Quickshell.execDetached(["sh", "-c", 'printf -- "- %s\\n" "$1" >> "$2"', "sh", arg, Generated.notesFile]);
            break;
        case "emoji":
            Niri.spawn([Generated.wlCopy, item.emoji]);
            break;
        case "clip":
            if (shift) {
                Quickshell.execDetached(["sh", "-c", 'printf "%s" "$1" | "$2" delete', "sh", item.line, Generated.cliphist]);
                clipEntries = clipEntries.filter(l => l !== item.line);
                return;
            }
            Quickshell.execDetached(["sh", "-c", 'printf "%s" "$1" | "$2" decode | "$3"', "sh", item.line, Generated.cliphist, Generated.wlCopy]);
            break;
        default:
            return;
        }
        Panels.close();
    }

    function move(delta) {
        if (items.length > 0)
            selected = (selected + delta + items.length) % items.length;
    }

    // Fixed size with a plain fade: resizing per keystroke made the panel jump.
    width: 600
    height: Theme.pad * 2 + input.implicitHeight + maxRows * rowHeight + footer.implicitHeight + Theme.gap * 2
    slideDistance: 0

    onOpenChanged: if (open) {
        input.text = Panels.launcherPrefix;
        Panels.launcherPrefix = "";
        input.forceActiveFocus();
        if (mode === "clip")
            cliphist.running = true;
    }
    onModeChanged: if (mode === "clip")
        cliphist.running = true
    onItemsChanged: selected = 0
    // The timer checks calcActive itself: here it can still hold the value from before this change.
    onArgChanged: calcDebounce.restart()

    Timer {
        id: calcDebounce
        interval: 120
        onTriggered: {
            if (qalc.running || !root.calcActive || root.arg === "")
                return;
            qalc.expr = root.arg;
            qalc.running = true;
        }
    }

    Process {
        id: qalc

        property string expr: ""

        command: [Generated.qalc, "-t", expr]
        stdout: StdioCollector {
            onStreamFinished: {
                root.calcFor = qalc.expr;
                root.calcResult = text.trim();
            }
        }
        // Input changed while qalc was busy: catch up on the latest expression.
        onExited: if (root.calcActive && root.arg !== expr)
            calcDebounce.restart()
    }

    Process {
        id: cliphist
        command: [Generated.cliphist, "list"]
        stdout: StdioCollector {
            onStreamFinished: root.clipEntries = text.split("\n").filter(l => l !== "")
        }
    }

    FileView {
        path: Generated.emojiFile
        onLoaded: root.emojis = text().split("\n").filter(l => l !== "").map(l => {
                const tab = l.indexOf("\t");
                return {
                    char: l.slice(0, tab),
                    name: l.slice(tab + 1).toLowerCase()
                };
            })
    }

    FileView {
        path: Quickshell.statePath("launcher.json")
        blockLoading: true
        printErrors: false // first run: no file yet
        onAdapterUpdated: writeAdapter()

        JsonAdapter {
            id: usage
            property var counts: ({})
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Theme.pad
        spacing: Theme.gap

        TextField {
            id: input
            Layout.fillWidth: true
            placeholderText: "Search apps  ·  = math  > command  + note  : emoji  ; clipboard"
            color: Theme.textStrong
            placeholderTextColor: Theme.textDim
            font.family: Theme.fontSans
            font.pixelSize: Theme.fontLg
            leftPadding: 44
            rightPadding: 12
            onVisibleChanged: if (visible)
                forceActiveFocus()
            background: Rectangle {
                implicitHeight: 44
                radius: Theme.radiusMd
                color: Theme.surfaceHighest

                Icon {
                    anchors.left: parent.left
                    anchors.leftMargin: 14
                    anchors.verticalCenter: parent.verticalCenter
                    icon: ({
                            "calc": "calculate",
                            "run": "terminal",
                            "note": "edit_note",
                            "emoji": "mood",
                            "clip": "content_paste"
                        })[root.mode] ?? "search"
                    size: 20
                    color: root.mode === "apps" ? Theme.textMuted : Theme.accent
                }
            }

            Keys.onPressed: event => {
                const ctrl = event.modifiers & Qt.ControlModifier;
                if (event.key === Qt.Key_Down || event.key === Qt.Key_Tab || (ctrl && (event.key === Qt.Key_J || event.key === Qt.Key_N)))
                    root.move(1);
                else if (event.key === Qt.Key_Up || event.key === Qt.Key_Backtab || (ctrl && (event.key === Qt.Key_K || event.key === Qt.Key_P)))
                    root.move(-1);
                else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter)
                    root.activate(root.selected, event.modifiers & Qt.ShiftModifier);
                else if (event.key === Qt.Key_Escape)
                    Panels.close();
                else
                    return;
                event.accepted = true;
            }
        }

        ListView {
            id: list
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            model: root.items
            currentIndex: root.selected
            boundsBehavior: Flickable.StopAtBounds

            delegate: Rectangle {
                id: row

                required property var modelData
                required property int index
                readonly property bool isSelected: root.selected === index

                width: ListView.view.width
                height: root.rowHeight
                radius: Theme.radiusMd
                color: isSelected ? Theme.accentSoft : "transparent"

                StateLayer {
                    onEntered: root.selected = row.index
                    onClicked: mouse => root.activate(row.index, mouse.modifiers & Qt.ShiftModifier)
                }

                RowLayout {
                    anchors.fill: parent
                    anchors.leftMargin: 10
                    anchors.rightMargin: 12
                    spacing: 12

                    Item {
                        implicitWidth: 28
                        implicitHeight: 28

                        IconImage {
                            anchors.fill: parent
                            visible: row.modelData.kind === "app"
                            source: visible ? Quickshell.iconPath(row.modelData.app.icon, "application-x-executable") : ""
                        }

                        StyledText {
                            anchors.centerIn: parent
                            visible: row.modelData.kind === "emoji"
                            text: row.modelData.emoji ?? ""
                            font.pixelSize: 22
                        }

                        Icon {
                            anchors.centerIn: parent
                            visible: row.modelData.kind !== "app" && row.modelData.kind !== "emoji"
                            icon: row.modelData.icon ?? ""
                            size: 22
                            fill: row.isSelected
                            color: row.isSelected ? Theme.accent : Theme.text
                        }
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 0

                        StyledText {
                            Layout.fillWidth: true
                            text: row.modelData.title
                            color: row.isSelected ? Theme.textStrong : Theme.text
                            font.pixelSize: row.modelData.kind === "calc" ? Theme.fontXl : Theme.fontMd
                            font.weight: row.isSelected ? Font.DemiBold : Font.Medium
                        }

                        StyledText {
                            Layout.fillWidth: true
                            visible: text !== ""
                            text: row.modelData.subtitle ?? ""
                            font.pixelSize: Theme.fontSm
                            font.weight: Font.Normal
                            color: row.isSelected ? Theme.accent : Theme.textMuted
                        }
                    }
                }
            }
        }

        StyledText {
            id: footer
            Layout.fillWidth: true
            Layout.leftMargin: 4
            text: "↑↓ select  ·  Enter open  ·  Esc close"
            font.pixelSize: Theme.fontXs
            color: Theme.textDim
        }
    }
}
