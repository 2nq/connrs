# Changelog

## 0.2.0 — 2026-07-06

### Added

- **Filtering** (`/`): incrementally narrow the list by process name, PID,
  remote IP, or remote port. `Enter` applies, `Esc` clears.
- **Sort modes** (`s`): cycle bandwidth → name → connection count. Name sort
  gives a stable view when rates shuffle the default ordering.
- **Pause** (`p`): freeze the display to read or copy without rows moving;
  `r` still forces a one-off refresh while paused.
- **Reverse DNS**: remote endpoints on expanded rows show their hostname,
  resolved asynchronously off the render path and cached (including negative
  results).

### Fixed

- **Selection scrolling drifted on long process lists.** The recorded line
  ranges assumed a blank line between process rows that the renderer never
  emitted, so "keep selection visible" scrolled one line further off per row.
- **Wrong process names after PID reuse.** The PID→name cache was never
  pruned; Windows recycles PIDs, so a new process could be labeled with a
  dead process's name in long-running sessions. Entries for departed PIDs
  are now dropped every poll.
- **Bandwidth silently stopped tracking on reused TCP 4-tuples.** EStats
  collection is enabled per TCB, but the enabled set was keyed by connection
  tuple and never cleaned, so a reused tuple skipped re-enablement on the new
  TCB. Departed connections are now pruned every poll.
- Unbounded memory growth in three long-lived maps (bandwidth enablement,
  PID name cache, UI expand/collapse state).
- Out-of-range guard for expand/collapse when the selection index is stale.

### Changed

- Removed dead code (`errBandwidthNotSupported`, unused `min`/`max` helpers
  shadowing Go builtins) and deprecated `lipgloss.Style.Copy()` calls.
- Regression tests added for all of the fixes above.

## 0.1.0 — 2026-05-15

Initial release: Windows TCP/UDP network monitor TUI with per-process
grouping and TCP bandwidth via IP Helper extended statistics.
