# Lan Trans

Lan Trans is a lightweight Windows LAN transfer tool. It runs a small background HTTP service and exposes a browser page for devices on the same local network to share files and text messages.

## Features

- Browser-only UI
- File upload, download, and deletion
- Text message sharing
- Automatic cleanup after 24 hours
- Default LAN-only access boundary
- Preferred port `8787` with fallback through `8807`
- Windows startup task install/uninstall
- No database and no external runtime

## Build

Install Go 1.22 or newer, then run:

```powershell
go build -o lan-trans.exe .
```

## Run

```powershell
.\lan-trans.exe serve
```

Open one of the URLs printed by the program, usually:

```text
http://<your-pc-lan-ip>:8787
```

Other devices on the same LAN can open that URL in a browser.

On first run, Windows Defender Firewall may ask whether to allow private network access. Allow private network access if you want phones or other computers on the LAN to connect.

## Commands

```powershell
.\lan-trans.exe serve
.\lan-trans.exe status
.\lan-trans.exe install-startup
.\lan-trans.exe uninstall-startup
```

`install-startup` creates a Windows scheduled task named `LanTrans` that starts the server when you log in.
The startup task launches the server through PowerShell with a hidden window so it can run quietly in the background.

## Data Directory

By default, files and metadata are stored in:

```text
%LOCALAPPDATA%\LanTrans
```

The directory contains:

```text
config.json
runtime.json
metadata.json
messages.json
files\
```

## Configuration

The default `config.json` is created on first run:

```json
{
  "listenHost": "0.0.0.0",
  "preferredPort": 8787,
  "portFallbackEnd": 8807,
  "retentionHours": 24,
  "cleanupIntervalMinutes": 10,
  "maxUploadMB": 2048,
  "allowCIDRs": [
    "127.0.0.1/32",
    "::1/128",
    "10.0.0.0/8",
    "172.16.0.0/12",
    "192.168.0.0/16"
  ]
}
```

## Notes

Lan Trans intentionally has no password in the first version. It is designed for trusted private LAN use and rejects requests outside local/private address ranges.
