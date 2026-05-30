# t9s — TUI for Talos Linux

A terminal UI for managing Talos Linux clusters, inspired by k9s. Built with Go + [bubbletea](https://github.com/charmbracelet/bubbletea).

## Requirements

- Go 1.22+
- `talosctl` installed and in `$PATH`
- A valid talosconfig (`~/.talos/config` or `$TALOSCONFIG`)

## Installation

```bash
git clone https://github.com/florianspk/t9s
cd t9s
go build -o t9s ./cmd/main.go
sudo mv t9s /usr/local/bin/
```

Or run directly:

```bash
go run ./cmd/main.go
```

## Usage

```bash
t9s [flags]

Flags:
  --talosconfig <path>   Path to talosconfig (default: $TALOSCONFIG or ~/.talos/config)
  --context <name>       Talos context to use (default: active context in talosconfig)
  --version              Print version and exit
```

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `?` | Toggle help overlay |
| `x` | Switch talos context (multi-cluster) |
| `Ctrl+C` | Quit |

### Node List (default view)

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` / `s` | Open services for selected node |
| `e` | Open extensions for selected node |
| `m` | View machine config for selected node |
| `d` | Stream dmesg (kernel logs) for selected node |
| `t` | View container metrics/stats for selected node |
| `U` | Upgrade Talos on selected node |
| `K` | Upgrade Kubernetes (from selected controlplane) |
| `r` | Force refresh node list |
| `q` | Quit |

### Services

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` / `l` | Stream logs for selected service |
| `Esc` / `q` | Back to node list |

### Logs / Dmesg

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `Esc` / `q` | Stop streaming, go back |

### Machine Config / Extensions / Metrics

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate / scroll |
| `Esc` / `q` | Back to node list |

### Upgrade Talos / Upgrade Kubernetes

| Key | Action |
|-----|--------|
| type | Enter image or version |
| `Enter` | Proceed to confirmation |
| `y` | Confirm and start upgrade |
| `n` / `Esc` | Cancel |
| `Esc` | Abort running upgrade |

### Context Switcher

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate contexts |
| `Enter` | Switch to selected context |
| `Esc` / `q` | Cancel |

## Views

| View | What it shows | Underlying command |
|------|---------------|--------------------|
| Nodes | All cluster members | `talosctl get members` |
| Services | Services on selected node | `talosctl get servicestatuses -n <node>` |
| Logs | Live log stream | `talosctl logs -n <node> -f <service>` |
| Machine Config | Active machine config YAML | `talosctl get machineconfig -n <node> -o yaml` |
| Extensions | Installed extensions | `talosctl get extensions -n <node>` |
| Dmesg | Kernel log stream | `talosctl dmesg -n <node> -f` |
| Metrics | Container CPU/memory stats | `talosctl stats -n <node>` |
| Upgrade Talos | Upgrade Talos on a node | `talosctl upgrade -n <node> --image <image>` |
| Upgrade K8s | Upgrade Kubernetes | `talosctl upgrade-k8s --to <version>` |

## Auto-refresh

- Node list and metrics refresh every **5 seconds** automatically.
- Log and dmesg streams are live (`-f` flag).
- All other views are loaded once; press `r` from the node list to refresh.

## Architecture

```
t9s/
├── cmd/main.go              # Entry point (CLI flags, program setup)
├── internal/
│   ├── config/config.go     # Talosconfig loading
│   ├── talos/
│   │   ├── types.go         # Data types (Node, Service, Extension…)
│   │   └── client.go        # talosctl subprocess wrappers
│   └── ui/
│       ├── app.go           # Main bubbletea model + lifecycle
│       ├── styles.go        # Lipgloss style definitions
│       ├── messages.go      # tea.Msg types
│       ├── keyrouter.go     # Keyboard routing
│       ├── nodelist.go      # Node list view
│       ├── services.go      # Services view
│       ├── logs.go          # Log streaming view
│       ├── machineconfig.go # Machine config view
│       ├── extensions.go    # Extensions view
│       ├── dmesg.go         # Dmesg streaming view
│       ├── metrics.go       # Metrics view
│       ├── upgrade.go       # Upgrade Talos/K8s view
│       └── contexts.go      # Context switcher view
├── go.mod
└── README.md
```

## Talos API Notes

- Uses `talosctl` subprocesses (not the machinery gRPC library) for simplicity and reliability.
- All JSON parsing uses ndjson output from `talosctl get ... -o json`.
- Streaming (logs, dmesg) uses goroutines with context cancellation — no goroutine leaks.
- Supports HA controlplane (multiple endpoints defined in talosconfig).
