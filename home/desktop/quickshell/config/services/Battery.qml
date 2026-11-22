pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Services.UPower

Singleton {
    id: root

    readonly property UPowerDevice device: UPower.displayDevice
    readonly property bool available: device?.isLaptopBattery ?? false
    readonly property real percent: device?.percentage ?? 0
    readonly property bool charging: device?.state === UPowerDeviceState.Charging || device?.state === UPowerDeviceState.PendingCharge
    readonly property bool full: device?.state === UPowerDeviceState.FullyCharged
    readonly property bool pluggedIn: !UPower.onBattery
    readonly property real watts: Math.abs(device?.changeRate ?? 0)
    readonly property bool low: !charging && percent <= 0.2
    readonly property bool critical: !charging && percent <= 0.1

    readonly property string icon: {
        if (charging)
            return "battery_android_bolt";
        if (full || percent >= 0.97)
            return "battery_android_full";
        if (critical)
            return "battery_android_alert";
        return "battery_android_" + Math.max(0, Math.min(6, Math.round(percent * 6)));
    }

    readonly property string timeLeft: {
        const s = charging ? device?.timeToFull : device?.timeToEmpty;
        if (!s || s <= 0)
            return "";
        const h = Math.floor(s / 3600);
        const m = Math.round((s % 3600) / 60);
        return (h > 0 ? h + "h " : "") + m + "m";
    }

    readonly property string status: {
        if (full)
            return "Fully charged";
        if (charging)
            return timeLeft ? "Charging · full in " + timeLeft : "Charging";
        if (pluggedIn)
            return "Plugged in";
        return timeLeft ? timeLeft + " remaining" : "On battery";
    }
}
