# Lan Flow — LAN File Flow

> Lightweight LAN file sharing — open a browser and go.

Lan Flow is a single-binary, zero-dependency Windows background service. Start it on your computer, and **any device on the same LAN** (phone, tablet, another PC) can upload/download files and share text messages through a browser. No app installs, no accounts, no cloud needed.

[中文版说明](README_cn.md)

---

## Quick Start

### Run

```powershell
.\lan-flow.exe serve
```

The terminal prints addresses like:

```
Lan Flow listening on 0.0.0.0:8787
URL: http://192.168.1.100:8787
URL: http://172.20.0.100:8787
URL: http://127.0.0.1:8787
```

Open `http://192.168.1.100:8787` (use whatever your terminal shows) on another device's browser.

### Check status

```powershell
.\lan-flow.exe status
```

### Auto-start on login

```powershell
.\lan-flow.exe install-startup
```

The service starts silently in the background every time you log in.

### Remove auto-start

```powershell
.\lan-flow.exe uninstall-startup
```

---

## Use Cases

| Scenario | How |
|----------|-----|
| 📁 Send a file to a coworker | Drag to upload, they download in browser — no USB, no WeChat |
| 📝 Share a snippet / link / code | Paste into the message box, others copy it instantly |
| 📱 Phone ↔ PC transfer | Open the address on your phone's browser, no cable needed |
| 🏠 Home devices | Run the server on your Windows desktop, access from laptop / iPad / phone |

---

## Data Directory

Files are stored in `%LOCALAPPDATA%\LanFlow`:

```
config.json     Configuration (port, CIDR whitelist, etc.)
runtime.json    Current runtime info
metadata.json   File index
messages.json   Message history
files/          Uploaded files
```

Uploads are **automatically cleaned up after 24 hours**.

---

## Security Notes

- **No password** (designed for trusted LANs only)
- **IP-restricted by default** — only localhost and private ranges (`10.x.x.x`, `172.16-31.x.x`, `192.168.x.x`)
- **Port fallback** — tries `8787` first, walks up to `8807` if busy
- **Single-instance guard** — won't start a second copy if one is already running
- ⚠️ **Do not expose directly to the internet**

---

## Specs

| Item | Value |
|------|-------|
| Platform | Windows (Go, single exe) |
| Default port | 8787 (range 8787–8807) |
| Max file size | 2048 MB |
| Max message length | 64 KB |
| Retention | 24 hours (auto-cleanup) |
| UI | Web browser |

---

## API Overview

```
GET    /                  Web UI
GET    /api/health        Health check
GET    /api/items         All files and messages
POST   /api/files         Upload a file
GET    /api/files/{id}    Download a file
DELETE /api/files/{id}    Delete a file
POST   /api/messages      Send a message
GET    /api/messages      List messages
DELETE /api/messages/{id} Delete a message
```

---

## Build from Source

Requires Go 1.22+:

```powershell
go build -o lan-flow.exe .
```

---

## Troubleshooting

### Port unexpectedly changed (e.g. 8788 instead of 8787)

If the server starts on a different port than expected, the preferred port (`8787`) is already in use.

**Common causes:**

1. **Another instance is already running** — Check with `.\lan-flow.exe status`. If one is running, don't start a second one.
2. **Stale startup task from a previous version** — If you renamed the binary or upgraded from an older version, the old startup task (`LanTrans`) may still be launching the old binary. Run `.\lan-flow.exe install-startup` again — it automatically removes the old task.
3. **Windows reserved port range** — Some Windows configurations (especially Hyper-V / WSL2) reserve ranges of ports. Run this to check:
   ```powershell
   netsh int ipv4 show excludedportrange protocol=tcp
   ```
   If `8787` is in the list, you can change the preferred port in `%LOCALAPPDATA%\LanFlow\config.json`.

---

## Design Docs

See [BASELINE.md](BASELINE.md) for the full development plan.

---

## License

MIT

---
