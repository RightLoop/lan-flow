# Lan Flow Development Baseline

## Goal

Lan Flow is a lightweight Windows background program that exposes a local network web page for peer-style file and message transfer between devices on the same LAN.

## Platform And Runtime

- Target platform: Windows
- Implementation language: Go
- Runtime form: single executable
- UI: browser-only web interface
- Default host: `0.0.0.0`
- Preferred port: `8787`
- Port fallback range: `8787-8807`
- Default retention: 24 hours
- Default maximum upload size: 2048 MB
- Default data directory: `%LOCALAPPDATA%\LanFlow`

## Security Boundary

There is no authentication in the first version because the tool is intended for trusted LAN use. The server still rejects requests outside local/private address ranges:

- `127.0.0.1/32`
- `::1/128`
- `10.0.0.0/8`
- `172.16.0.0/12`
- `192.168.0.0/16`

## Required Features

- Background HTTP service
- Browser home page
- File upload
- File list
- File download
- File deletion
- Text message creation
- Text message list
- Text message deletion
- Automatic cleanup after 24 hours
- Config file
- Port fallback
- Runtime status file with actual port
- Windows startup install/uninstall commands
- README usage guide

## Deferred Features

- Password login
- Native mobile app
- Public internet access
- Multi-PC synchronization
- WebSocket push
- Windows Service mode
- Folder upload
- Resumable transfers
- Transfer throttling
- Complex permissions

## Commands

```text
lan-flow.exe serve
lan-flow.exe status
lan-flow.exe install-startup
lan-flow.exe uninstall-startup
```

## API

```text
GET    /
GET    /api/health
GET    /api/items
POST   /api/files
GET    /api/files/{id}
DELETE /api/files/{id}
GET    /api/messages
POST   /api/messages
DELETE /api/messages/{id}
```

