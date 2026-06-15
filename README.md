<div align="center">

<img src="assets/logo.png" alt="t9s" width="350"/>

# t9s — Talos Linux CLI To Manage Your Clusters In Style!

**t9s provides a terminal UI to interact with your [Talos Linux](https://www.talos.dev) clusters.**
The aim of this project is to make it easier to navigate, observe and manage your Talos nodes in the wild. t9s continually watches your cluster for changes and offers subsequent commands to interact with your observed resources.

*Think [k9s](https://k9scli.io), but for Talos.*

<br/>

[![Go Report Card](https://goreportcard.com/badge/github.com/florianspk/t9s)](https://goreportcard.com/report/github.com/florianspk/t9s)
[![GitHub Release](https://img.shields.io/github/v/release/florianspk/t9s)](https://github.com/florianspk/t9s/releases)
[![License](https://img.shields.io/badge/license-Source--Available-blue)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/florianspk/t9s)](go.mod)
[![Downloads](https://img.shields.io/github/downloads/florianspk/t9s/total)](https://github.com/florianspk/t9s/releases)

<br/>

![t9s demo](assets/t9s.gif)

</div>

---

## Features

- 🖥️ **Full-screen responsive layout** — adapts to any terminal size, columns expand with the window
- 📋 **Node list** — Talos version, Kubernetes version, role and status at a glance
- 📡 **Live streaming** — services, logs and dmesg with an interactive ▶ cursor
- 🔍 **Per-node resource views** — disks, processes, containers, network addresses
- 📊 **Metrics** — CPU/RAM stats with delta, auto-refreshed every 5s
- 📄 **Machine config** — read-only YAML viewer
- 🧩 **Extensions** — installed list + Siderolabs catalog browser (requires `crane`)
- ⬆️ **Upgrades** — Talos and Kubernetes, with version pre-fill and `--preserve` toggle
- 🩺 **Health** — streaming cluster health checks
- 🔀 **Multi-context** — switch talosconfig context at runtime (`x`)
- ⚡ **Search** — real-time filtering in every list view (`/`)
- 📐 **Wrap mode** — toggle line wrapping for wide content (`w`)
- 🚨 **Version check** — warns when talosctl client/server versions diverge

---

## Installation

t9s is available on Linux and macOS.

### Homebrew (macOS / Linux)

```bash
brew tap florianspk/tap
brew install --cask t9s
```

### Packages & binaries

Download from the [GitHub Releases](https://github.com/florianspk/t9s/releases) page:

- `.tar.gz` archives for **linux** / **darwin** (amd64, arm64)
- `.deb` / `.rpm` / `.apk` packages for Linux

### From source

Requires **Go 1.22+**.

```bash
git clone https://github.com/florianspk/t9s
cd t9s
go build -o t9s ./cmd/main.go
sudo mv t9s /usr/local/bin/
```

---

## Prerequisites

| Requirement | Notes |
|---|---|
| `talosctl` **≥ 1.5** | must be in `$PATH` |
| A valid talosconfig | `~/.talos/config` or `$TALOSCONFIG` |
| `crane` | optional — only for the extension catalog view (`C`) |

### talosctl compatibility

| talosctl | Status |
|----------|--------|
| **1.7.x** | ✅ Tested |
| **1.6.x** | ✅ Tested |
| **1.5.x** | ✅ Minimum supported |
| < 1.5 | ❌ `talosctl get disks` / `get extensions` not available |

> [!NOTE]
> Your talosctl client version should match your cluster version (±1 minor).
> t9s warns you when they diverge.

<details>
<summary><b>talosctl commands used under the hood</b></summary>

| Feature | Command | Available since |
|---------|---------|-----------------|
| Node list | `talosctl get members -o json` | 1.0 |
| Services | `talosctl services` | 1.0 |
| Logs | `talosctl logs -f` | 1.0 |
| Dmesg | `talosctl dmesg -f` | 1.0 |
| Machine config | `talosctl get machineconfig -o yaml` | 1.0 |
| Edit config | `talosctl apply-config --mode auto` | 1.0 |
| Patch config | `talosctl patch machineconfig --patch @file` | 1.2 |
| Addresses | `talosctl get addresses -o json` | 1.2 |
| Extensions | `talosctl get extensions -o json` | 1.3 |
| K8s version | `talosctl get kubeletspec -o json` | 1.3 |
| Disks | `talosctl get disks -o json` | **1.5** |
| Processes | `talosctl processes` | 1.0 |
| Containers | `talosctl containers` | 1.0 |
| Stats | `talosctl stats` | 1.0 |
| Health | `talosctl health` | 1.0 |
| Upgrade Talos | `talosctl upgrade` | 1.0 |
| Upgrade K8s | `talosctl upgrade-k8s` | 1.0 |
| Reboot / Shutdown | `talosctl reboot` / `shutdown` | 1.0 |

</details>

---

## Usage

```bash
# Launch with your default talosconfig
t9s

# Specify a talosconfig and context
t9s --talosconfig ~/.talos/config --context my-cluster

# Print version
t9s --version
```

---

## Key Bindings

t9s uses aliases to navigate most Talos resources — hit `?` at any time for the in-app help overlay.

### Global

| Key | Action |
|-----|--------|
| <kbd>?</kbd> | Help overlay |
| <kbd>/</kbd> | Search / filter |
| <kbd>w</kbd> | Toggle wrap mode |
| <kbd>x</kbd> | Switch talos context |
| <kbd>Ctrl</kbd>+<kbd>C</kbd> | Quit |

### Node list

| Key | Action | | Key | Action |
|-----|--------|-|-----|--------|
| <kbd>↑</kbd><kbd>↓</kbd> / <kbd>j</kbd><kbd>k</kbd> | Navigate | | <kbd>t</kbd> | Metrics |
| <kbd>Enter</kbd> / <kbd>s</kbd> | Services | | <kbd>p</kbd> | Processes |
| <kbd>e</kbd> | Extensions | | <kbd>c</kbd> | Containers |
| <kbd>C</kbd> | Extension catalog | | <kbd>a</kbd> | Network addresses |
| <kbd>m</kbd> | Machine config | | <kbd>i</kbd> | Disks |
| <kbd>d</kbd> | Dmesg | | <kbd>H</kbd> | Cluster health |
| <kbd>U</kbd> | Upgrade Talos | | <kbd>R</kbd> / <kbd>S</kbd> | Reboot / Shutdown |
| <kbd>K</kbd> | Upgrade Kubernetes | | <kbd>r</kbd> | Refresh |

### Logs / Dmesg / Health

| Key | Action |
|-----|--------|
| <kbd>↑</kbd><kbd>↓</kbd> | Move cursor |
| <kbd>PgUp</kbd> / <kbd>PgDn</kbd> | Half-page scroll |
| <kbd>g</kbd> / <kbd>G</kbd> | Top / bottom |
| <kbd>Esc</kbd> / <kbd>q</kbd> | Back |

### Upgrade

| Key | Action |
|-----|--------|
| type | Enter image or version (pre-filled with current) |
| <kbd>p</kbd> | Toggle `--preserve` (default on — required for single-node etcd) |
| <kbd>Enter</kbd> | Confirm |
| <kbd>y</kbd> / <kbd>n</kbd> | Confirm / cancel |
| <kbd>Esc</kbd> | Back (upgrade keeps running in background) |

---

## Views

| View | Key | What it shows |
|------|-----|---------------|
| Nodes | *default* | Members — Talos + K8s version, role, status |
| Services | <kbd>s</kbd> | Service state and health |
| Logs | <kbd>l</kbd> | Live service log stream |
| Dmesg | <kbd>d</kbd> | Live kernel log stream |
| Machine Config | <kbd>m</kbd> | Machine config YAML |
| Extensions | <kbd>e</kbd> | Installed Talos extensions |
| Ext. Catalog | <kbd>C</kbd> | Available extensions from the Siderolabs registry |
| Metrics | <kbd>t</kbd> | CPU/RAM per container with delta |
| Processes | <kbd>p</kbd> | Running processes sorted by memory |
| Containers | <kbd>c</kbd> | containerd containers (system + k8s namespaces) |
| Addresses | <kbd>a</kbd> | Network interfaces and addresses |
| Disks | <kbd>i</kbd> | Block devices — model, serial, type, size |
| Health | <kbd>H</kbd> | Cluster health checks (streaming) |
| Upgrade Talos | <kbd>U</kbd> | Upgrade with pre-filled installer image |
| Upgrade K8s | <kbd>K</kbd> | Upgrade with pre-filled version |

---

## Architecture

```
t9s/
├── cmd/main.go               # Entry point, CLI flags, bubbletea setup
├── internal/
│   ├── config/config.go      # Talosconfig loader
│   ├── talos/
│   │   ├── types.go          # Data types (Node, Service, DiskInfo…)
│   │   └── client.go         # talosctl subprocess wrappers
│   └── ui/
│       ├── app.go            # bubbletea Model: Init / Update / View
│       ├── styles.go         # Lipgloss palette and styles
│       ├── messages.go       # tea.Msg types
│       ├── keyrouter.go      # Global key dispatch
│       ├── hints.go          # Context-sensitive hint bar
│       └── <view>.go         # One file per view
├── hack/vagrant/             # VirtualBox test cluster
├── assets/                   # Logo and visual assets
├── go.mod
└── LICENSE
```

**Design notes**

- Wraps `talosctl` as a subprocess — no gRPC dependency, authentication is inherited automatically
- Responsive column widths computed from the terminal width at render time
- Backward line-counting guarantees the cursor is always visible in wrap mode
- Goroutine + channel streaming with context cancellation — no goroutine leaks

---

## Development — local test cluster

Spin up a throwaway QEMU cluster to test upgrades safely:

```bash
# Download Talos assets
mkdir -p _out
curl -L https://github.com/siderolabs/talos/releases/download/v1.7.0/vmlinuz-amd64 -o _out/vmlinuz-amd64
curl -L https://github.com/siderolabs/talos/releases/download/v1.7.0/initramfs-amd64.xz -o _out/initramfs-amd64.xz

# Create QEMU cluster (requires root for the CNI bridge)
sudo -E env TALOSCONFIG=~/.talos/t9s-dev.yaml talosctl cluster create \
  --provisioner qemu --name t9s-dev --controlplanes 1 --workers 1 \
  --vmlinuz-path _out/vmlinuz-amd64 --initrd-path _out/initramfs-amd64.xz \
  --talosconfig ~/.talos/t9s-dev.yaml --skip-kubeconfig

t9s --talosconfig ~/.talos/t9s-dev.yaml
```

A VirtualBox alternative lives in [`hack/vagrant/`](hack/vagrant/).

---

## Contributing

Issues and pull requests are welcome! Note that per the license, all modifications must be contributed back to this repository.

---

## License

[Non-Commercial Source-Available License](LICENSE) — free for personal and open-source use.
Commercial use is prohibited. All modifications must be contributed back to this repository.

---

<div align="center">

Made with ❤️ for the Talos community

</div>
