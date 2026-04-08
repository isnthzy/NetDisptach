---
name: linux-nic-detection-fix
description: Fix Linux NIC detection for auto-selecting bind address
type: design
created: 2026-04-08
---

# Linux NIC Detection Fix

## Problem Statement

Linux users report that the proxy starts normally but forwarding fails. Root cause analysis indicates that the `autoSelectBindAddress()` function uses Windows-centric NIC name matching patterns that don't work with Linux interface naming conventions.

## Root Cause

`cmd/netdispatch/main.go:autoSelectBindAddress()` matches NIC names against:
- Windows: `eth`, `以太`, `网线` (Ethernet)
- Windows: `wlan`, `wifi` (Wireless)

Linux uses predictable network interface names:
- Ethernet: `enp0s3`, `ens33`, `eno1`, `enx...`
- Wireless: `wlp2s0`, `wlo1`, `wlx...`

These Linux patterns don't match the current detection logic.

## Solution

Expand NIC name matching patterns to support both Windows and Linux naming conventions.

### Changes to `autoSelectBindAddress()`

Add Linux-specific NIC name patterns:

**Ethernet patterns** (cross-platform):
- Windows: `以太`, `网线`, `ethernet`
- Linux: `eth`, `enp`, `ens`, `eno`, `enx`

**Wireless patterns** (cross-platform):
- Windows: `wlan`, `wifi`
- Linux: `wlp`, `wlo`, `wlx`, `wlan`

### Implementation

```go
// Ethernet patterns (cross-platform)
isEthernet := strings.Contains(nameLower, "eth") ||
    strings.Contains(nameLower, "enp") ||
    strings.Contains(nameLower, "ens") ||
    strings.Contains(nameLower, "eno") ||
    strings.Contains(nameLower, "enx") ||
    strings.Contains(nameLower, "以太") ||
    strings.Contains(nameLower, "网线") ||
    strings.Contains(nameLower, "ethernet")

// Wireless patterns (cross-platform)
isWireless := strings.Contains(nameLower, "wlan") ||
    strings.Contains(nameLower, "wlp") ||
    strings.Contains(nameLower, "wlo") ||
    strings.Contains(nameLower, "wlx") ||
    strings.Contains(nameLower, "wifi") ||
    strings.Contains(nameLower, "无线")
```

### Priority Logic (unchanged)

1. Ethernet (first match)
2. WLAN (first match)
3. Default NIC (if available)
4. `0.0.0.0` (fallback)

### Testing

1. Build on Linux and verify NIC detection logs
2. Test forwarding with auto-selected bind address
3. Verify backward compatibility on Windows

## Files Changed

- `cmd/netdispatch/main.go`: Update `autoSelectBindAddress()` function

## Risk Assessment

- **Low risk**: Simple string matching change
- **Backward compatible**: Windows patterns preserved
- **No breaking changes**: Behavior same for Windows users
