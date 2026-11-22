pragma Singleton

import QtQuick
import Quickshell
import Quickshell.Io
import Quickshell.Services.Pam

// Session lock state and authentication (the surfaces are modules/lock/LockScreen.qml).
// Password and fingerprint run as two separate PAM stacks at the same time, so typing
// a password never waits on the reader: /etc/pam.d/quickshell-lock (password only)
// and /etc/pam.d/quickshell-fprint (fingerprint only), both written by the NixOS config.
Singleton {
    id: root

    property bool locked: false
    // The compositor confirmed the lock surfaces cover every output.
    property bool secure: false

    // Typed on whichever output has focus, shared by all of them.
    property string password: ""
    property string status: ""
    readonly property bool checking: passwd.active

    // "off": no reader, nothing enrolled or not started; "idle": timed out, any input
    // rearms it; "waiting": touch now; "retry": the last touch did not match.
    property string fingerprint: "off"
    property int fprintMisses: 0
    property real fprintStarted: 0
    property bool fprintMismatch: false

    signal rejected

    // Marker that survives a Quickshell crash: the compositor keeps the session locked
    // without a client, so a restarted shell has to take the lock again.
    readonly property string marker: Quickshell.env("XDG_RUNTIME_DIR") + "/quickshell-locked"

    function lock() {
        if (locked)
            return;
        password = "";
        status = "";
        fprintMisses = 0;
        locked = true;
        Quickshell.execDetached(["touch", marker]);
        startFingerprint();
    }

    function unlock() {
        fprintRetry.stop();
        fprint.abort();
        passwd.abort();
        locked = false;
        secure = false;
        password = "";
        status = "";
        fingerprint = "off";
        Quickshell.execDetached(["rm", "-f", marker]);
    }

    function submit() {
        if (!locked || passwd.active || password === "")
            return;
        status = "";
        passwd.start();
    }

    // Any key, pointer motion or resume: rearm a reader that timed out.
    function wake() {
        if (locked && fingerprint === "idle") {
            fprintMisses = 0;
            startFingerprint();
        }
    }

    function startFingerprint() {
        if (!locked || fprint.active)
            return;
        fprintMismatch = false;
        fprintStarted = Date.now();
        fingerprint = fprint.start() ? "waiting" : "off";
    }

    PamContext {
        id: passwd
        configDirectory: "/etc/pam.d"
        config: "quickshell-lock"

        onPamMessage: {
            if (responseRequired)
                respond(root.password);
            else if (messageIsError)
                root.status = message;
        }
        onCompleted: result => {
            if (result === PamResult.Success) {
                root.unlock();
                return;
            }
            root.password = "";
            root.status = "Wrong password";
            root.rejected();
        }
        onError: root.status = "Authentication error"
    }

    PamContext {
        id: fprint
        configDirectory: "/etc/pam.d"
        config: "quickshell-fprint"

        // pam_fprintd reports a non-matching touch as an error message and keeps going.
        onPamMessage: if (messageIsError) {
            root.fprintMismatch = true;
            root.fingerprint = "retry";
            root.rejected();
        } else if (root.fingerprint === "retry") {
            root.fingerprint = "waiting";
        }
        onCompleted: result => {
            if (result === PamResult.Success) {
                root.unlock();
                return;
            }
            if (!root.locked)
                return;
            // Instant failure: no reader or no enrolled finger, stay password-only.
            if (Date.now() - root.fprintStarted < 2000) {
                root.fingerprint = "off";
                return;
            }
            // Out of tries after misses: someone is touching it, rearm a few times.
            // Plain timeout: go idle so the reader isn't polled all night.
            if (root.fprintMismatch && ++root.fprintMisses < 4) {
                root.fingerprint = "retry";
                fprintRetry.restart();
            } else {
                root.fingerprint = "idle";
            }
        }
        onError: root.fingerprint = "off"
    }

    Timer {
        id: fprintRetry
        interval: 600
        onTriggered: root.startFingerprint()
    }

    // Re-take the lock after a crash or restart while locked.
    FileView {
        path: root.marker
        blockLoading: true
        printErrors: false
        onLoaded: root.lock()
    }

    IpcHandler {
        target: "lock"

        function lock(): void {
            root.lock();
        }

        // Polled by the lock command so suspend waits until the screen is covered.
        function isSecure(): bool {
            return root.secure;
        }

        function wake(): void {
            root.wake();
        }
    }
}
