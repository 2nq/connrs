# connrs

A terminal-based network monitor for Windows. Shows live TCP and UDP connections grouped by process, with per-connection bandwidth (rx/tx) tracked via Windows TCP extended statistics.

```
CONNRS  v0.1  ●  3 procs  14 conns  6 tcp tracked   admin ✓   1s
────────────────────────────────────────────────────────────────
  chrome.exe  (PID 4821)       ↑ 142 B/s  ↓ 2.1 kB/s
  msedge.exe  (PID 9042)       ↑  0 B/s   ↓  84 B/s
  svchost.exe (PID 1248)       ↑ 60 B/s   ↓ 120 B/s
```

## Features

- Live TCP/UDP connection list refreshed every second
- Per-process grouping with expand/collapse
- TCP rx/tx bandwidth using Windows IP Helper eStats
- Admin-awareness: warns when running without elevation (reduces visibility)
- Keyboard-driven TUI with viewport scrolling

## Platform support

| Platform | Run | Cross-compile from |
|----------|-----|--------------------|
| Windows  | yes | Windows, Linux, macOS |
| Linux    | no  | — |
| macOS    | no  | — |

`connrs` uses Windows-only APIs (`iphlpapi.dll` eStats, WMI) that have no equivalent on Linux or macOS. The binary must **run on Windows**. You can **build** it from any platform with a Go toolchain.

## Requirements

- Go 1.26+
- Windows 10 / Server 2019 or later (to run)
- Administrator privileges recommended (for full connection visibility and bandwidth data)

## Build

### On Windows

```powershell
go build .
.\connrs.exe
```

Or run without a build step:

```powershell
go run .
```

### Cross-compile from Linux / macOS

```bash
GOOS=windows GOARCH=amd64 go build -o connrs.exe .
```

Copy `connrs.exe` to a Windows machine and run it there.

## Usage

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` / `Space` | Expand / collapse process |
| `r` | Force immediate refresh |
| `?` | Toggle help |
| `q` | Quit |

Run as Administrator for complete TCP bandwidth data. When running without elevation, a warning banner is shown and some connections / counters may be missing.

## Bandwidth notes

- TCP rx/tx values update after the second poll (snapshot diffing needs two samples)
- UDP connections are listed but bandwidth is always shown as unavailable — Windows does not expose per-endpoint UDP byte counters through IP Helper
- Idle TCP connections correctly show `0 B/s`; the important signal is the `tcp tracked` count in the header

## License

MIT
