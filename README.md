# NetMeter - macOS menu bar app

Go app for macOS that shows the network traffic used since you connected to the current network/interface in the menu bar.

## How to run

On a Mac with Go installed:

```bash
./build.sh
open NetMeter.app
```

If macOS blocks the app:

```bash
xattr -dr com.apple.quarantine NetMeter.app
open NetMeter.app
```

## What it shows

In the menu bar:

```text
↓ 123 MB ↑ 45 MB
```

When clicked:

- active interface
- Wi-Fi network/SSID when available
- time since the baseline
- downloaded data
- uploaded data
- current speed
- Reset counter
- Quit

## Note

The counter resets when the active interface or detected SSID changes. For Ethernet, the SSID appears as `—`.
