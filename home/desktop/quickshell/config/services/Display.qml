pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Services.UPower
import qs.common

// Refresh rate of the built-in panel. Auto runs it at the lowest rate on
// battery and the highest on AC; a fixed pick holds until changed and is
// remembered across restarts. modules/power/RefreshRate.qml applies it.
Singleton {
    id: root

    readonly property string output: Generated.internalOutput

    // Rates (mHz, ascending) the panel offers at its native resolution.
    property var rates: []
    property int width: 0
    property int height: 0
    readonly property bool available: rates.length > 1

    // 0 = auto, otherwise a rate from `rates`.
    readonly property int choice: rates.includes(saved.refresh) ? saved.refresh : 0

    readonly property int activeRate: {
        if (rates.length === 0)
            return 0;
        if (choice !== 0)
            return choice;
        return UPower.onBattery ? rates[0] : rates[rates.length - 1];
    }

    readonly property string mode: activeRate > 0 ? width + "x" + height + "@" + (activeRate / 1000).toFixed(3) : ""

    function hz(rate) {
        return Math.round(rate / 1000);
    }

    function setChoice(rate) {
        saved.refresh = rate;
    }

    Process {
        running: true
        command: ["niri", "msg", "--json", "outputs"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    const modes = JSON.parse(text)[root.output]?.modes ?? [];
                    const native = modes.find(m => m.is_preferred) ?? modes[0];
                    if (!native)
                        return;
                    root.width = native.width;
                    root.height = native.height;
                    root.rates = [...new Set(modes.filter(m => m.width === native.width && m.height === native.height).map(m => m.refresh_rate))].sort((a, b) => a - b);
                } catch (e) {
                    console.warn("display: bad outputs json", e);
                }
            }
        }
    }

    FileView {
        path: Quickshell.statePath("display.json")
        blockLoading: true
        printErrors: false // first run: no file yet, defaults apply
        onAdapterUpdated: writeAdapter()

        JsonAdapter {
            id: saved
            property int refresh: 0
        }
    }
}
