<img src="assets/logo.png" alt="t9s" width="200"/>

# t9s — TUI for Talos Linux

A terminal UI for managing Talos Linux clusters, inspired by k9s.
Built with Go, [bubbletea](https://github.com/charmbracelet/bubbletea) and [lipgloss](https://github.com/charmbracelet/lipgloss).

![t9s demo](assets/t9s.gif)

---

## Features

- **Full-screen responsive layout** — adapts to any terminal size, columns expand with the window
- **Node list** with Talos version, Kubernetes version, role and status
- **Services, Logs, Dmesg** — live streaming with interactive ▶ cursor
- **Disks, Processes, Containers, Addresses** — per-node resource views
- **Metrics** — CPU/RAM stats with delta (auto-refreshes every 5s)
- **Machine config** — read-only YAML viewer
- **Extensions** — installed list + catalog browser (requires `crane`)
- **Upgrades** — Talos and Kubernetes with version pre-fill and `--preserve` toggle
- **Health** — cluster health streaming
- **Multi-context** — switch talosconfig context at runtime (`x`)
- **Search** — real-time filter in list views (`/`)
- **Wrap mode** — toggle line wrapping for wide content (`w`)
- **Version check** — warns when talosctl client/server versions diverge

---

## Requirements

- Go 1.22+ _(build from source only)_
- `talosctl` **≥ 1.5** in `$PATH`
- A valid talosconfig (`~/.talos/config` or `$TALOSCONFIG`)
- `crane` in `$PATH` — only required for the extension catalog view (`C`)

### talosctl version compatibility

| talosctl | Status |
|----------|--------|
| **1.5.x** | ✅ Minimum supported |
| **1.6.x** | ✅ Tested |
| **1.7.x** | ✅ Tested |
| < 1.5 | ❌ `talosctl get disks` and `talosctl get extensions` not available |

> **Note:** the talosctl client version should match your cluster version (±1 minor).
> t9s warns you when they diverge.

#### Commands used

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
| **Disks** | `talosctl get disks -o json` | **1.5** |
| Processes | `talosctl processes` | 1.0 |
| Containers | `talosctl containers` | 1.0 |
| Stats | `talosctl stats` | 1.0 |
| Health | `talosctl health` | 1.0 |
| Upgrade Talos | `talosctl upgrade` | 1.0 |
| Upgrade K8s | `talosctl upgrade-k8s` | 1.0 |
| Reboot / Shutdown | `talosctl reboot / shutdown` | 1.0 |

---

## Installation

**Homebrew (macOS / Linux)**

```bash
brew tap florianspk/tap
brew install --cask t9s
```

**Binaries** — download from [GitHub Releases](https://github.com/florianspk/t9s/releases) (`.tar.gz` for linux/darwin, `.deb`/`.rpm`/`.apk` for Linux).

**From source**

```bash
git clone https://github.com/florianspk/t9s
cd t9s
go build -o t9s ./cmd/main.go
sudo mv t9s /usr/local/bin/
```

---

## Usage

```
t9s [--talosconfig <path>] [--context <name>] [--version]
```

---

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `?` | Help overlay |
| `/` | Search / filter |
| `w` | Toggle wrap mode |
| `x` | Switch talos context |
| `Ctrl+C` | Quit |

### Node List

| Key | Action |
|-----|--------|
| `↑↓` / `jk` | Navigate |
| `Enter` / `s` | Services |
| `e` | Extensions |
| `C` | Extension catalog |
| `m` | Machine config |
| `d` | Dmesg |
| `t` | Metrics |
| `p` | Processes |
| `c` | Containers |
| `a` | Network addresses |
| `i` | Disks |
| `H` | Cluster health |
| `U` | Upgrade Talos |
| `K` | Upgrade Kubernetes |
| `R` / `S` | Reboot / Shutdown node |
| `r` | Refresh |

### Logs / Dmesg / Health

| Key | Action |
|-----|--------|
| `↑↓` | Move cursor |
| `PgUp` / `PgDn` | Half-page scroll |
| `g` / `G` | Top / bottom |
| `Esc` / `q` | Back |

### Upgrade

| Key | Action |
|-----|--------|
| type | Enter image or version (pre-filled with current) |
| `p` | Toggle `--preserve` (default on — required for single-node etcd) |
| `Enter` | Confirm |
| `y` / `n` | Confirm / cancel |
| `Esc` | Back (upgrade keeps running in background) |

---

## Views

| View | Key | What it shows |
|------|-----|---------------|
| Nodes | default | Members — Talos + K8s version, role, status |
| Services | `s` | Service state and health |
| Logs | `l` | Live service log stream |
| Dmesg | `d` | Live kernel log stream |
| Machine Config | `m` | Machine config YAML |
| Extensions | `e` | Installed Talos extensions |
| Ext. Catalog | `C` | Available extensions from Siderolabs registry |
| Metrics | `t` | CPU/RAM per container with delta |
| Processes | `p` | Running processes sorted by memory |
| Containers | `c` | containerd containers (system + k8s namespaces) |
| Addresses | `a` | Network interfaces and addresses |
| Disks | `i` | Block devices — model, serial, type, size |
| Health | `H` | Cluster health checks (streaming) |
| Upgrade Talos | `U` | Upgrade Talos with pre-filled installer image |
| Upgrade K8s | `K` | Upgrade Kubernetes with pre-filled version |

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

**Design notes:**
- Uses `talosctl` as a subprocess — no gRPC dependency, inherits authentication automatically
- Responsive column widths computed from `app.width` at render time
- Backward line-counting guarantees the cursor is always visible in wrap mode
- Goroutine + channel streaming with context cancellation — no goroutine leaks

---

## Local test cluster (upgrades)

```bash
# Download Talos assets
mkdir -p _out
curl -L https://github.com/siderolabs/talos/releases/download/v1.7.0/vmlinuz-amd64 -o _out/vmlinuz-amd64
curl -L https://github.com/siderolabs/talos/releases/download/v1.7.0/initramfs-amd64.xz -o _out/initramfs-amd64.xz

# Create QEMU cluster (requires root for CNI bridge)
sudo -E env TALOSCONFIG=~/.talos/t9s-dev.yaml talosctl cluster create \
  --provisioner qemu --name t9s-dev --controlplanes 1 --workers 1 \
  --vmlinuz-path _out/vmlinuz-amd64 --initrd-path _out/initramfs-amd64.xz \
  --talosconfig ~/.talos/t9s-dev.yaml --skip-kubeconfig

t9s --talosconfig ~/.talos/t9s-dev.yaml
```

A VirtualBox alternative is in [`hack/vagrant/`](hack/vagrant/).

---

## License

[Non-Commercial Source-Available License](LICENSE) — free for personal and open-source use.  
Commercial use is prohibited. All modifications must be contributed back to this repository.
