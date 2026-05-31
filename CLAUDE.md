<!-- project-context -->
# t9s — Project Context (for new agents)

## What is t9s
TUI for managing Talos Linux clusters, inspired by k9s. Built in Go with bubbletea v1.3.x + lipgloss v1.1.x.  
Repo: `github.com/florianspk/t9s` — binary: `./cmd/main.go` → `t9s`  
Target: Talos Linux v1.6.x+, requires `talosctl` in `$PATH`, optional `crane` for extension catalog.

## Architecture — single-model, subprocess approach
- **No gRPC**: uses `talosctl` as a subprocess — inherits auth from `~/.talos/config` automatically
- **Single `App` struct** (not sub-models): all state in `internal/ui/app.go`
- **Views**: controlled by `AppState` enum; one `.go` file per view in `internal/ui/`
- **Streaming** (logs, dmesg, health, upgrade): goroutines + buffered channels + `waitForX` tea.Cmd pattern with context cancellation (no goroutine leaks)
- **Responsive layout**: column widths computed from `app.width` at render time
- **Wrap mode**: backward line-counting to keep cursor always visible

## File map
```
cmd/main.go                    CLI flags, bubbletea.NewProgram setup, version injection
internal/config/config.go      talosconfig loader (~/.talos/config or $TALOSCONFIG)
internal/talos/types.go        Data types: Node, Service, DiskInfo, ContainerInfo, ProcessInfo, AddressInfo, …
internal/talos/client.go       talosctl subprocess wrappers, parseNDJSON generic, parseContainerLines
internal/ui/app.go             App struct + Init/Update/View (1100 lines)
internal/ui/keyrouter.go       Global key dispatch by state
internal/ui/messages.go        tea.Msg types (MsgNodes, MsgServices, MsgLogLine, …)
internal/ui/styles.go          Lipgloss palette and shared styles
internal/ui/hints.go           Context-sensitive hint bar (bottom of screen)
internal/ui/nodelist.go        renderNodeList
internal/ui/services.go        renderServices
internal/ui/logs.go            renderLogs + streaming
internal/ui/dmesg.go           renderDmesg + streaming
internal/ui/machineconfig.go   renderMachineConfig (YAML viewer)
internal/ui/extensions.go      renderExtensions
internal/ui/extcatalog.go      renderExtCatalog (requires crane)
internal/ui/metrics.go         renderMetrics (auto-refresh 5s)
internal/ui/upgrade.go         renderUpgradeTalos / renderUpgradeK8s + runStreaming
internal/ui/health.go          renderHealth + streaming
internal/ui/contexts.go        renderContextSwitcher
internal/ui/helpview.go        renderHelp overlay
internal/ui/processes.go       renderProcesses
internal/ui/containers.go      renderContainers
internal/ui/disks.go           renderDisks
internal/ui/addresses.go       renderAddresses
internal/ui/search.go          search/filter logic for list views
```

## AppState enum (all views)
```go
StateNodeList, StateServices, StateLogs, StateMachineConfig, StateExtensions,
StateExtCatalog, StateDmesg, StateMetrics, StateUpgradeTalos, StateUpgradeK8s,
StateContextSwitcher, StateDisks, StateProcesses, StateContainers, StateAddresses,
StateHealth, StateHelp
```

## Layout constants
```go
headerBaseH = 3  // topBar + resourceLine + separator (without hints)
footerH     = 2  // separator + status line
```

## Key features implemented
- Full-screen responsive layout (columns expand with terminal width)
- Node list: Talos version, K8s version, role, status
- Services, Logs, Dmesg — live streaming with ▶ cursor
- Disks, Processes, Containers, Addresses — per-node views
- Metrics — CPU/RAM with delta, auto-refresh every 5s
- Machine config — read-only YAML viewer
- Extensions — installed list + catalog browser (crane)
- Upgrades — Talos & K8s with version pre-fill, --preserve toggle
- Health — cluster health streaming
- Multi-context — switch talosconfig context at runtime (`x`)
- Search — real-time filter in list views (`/`)
- Wrap mode — toggle line wrapping (`w`)
- Version check — warns when talosctl client/server versions diverge

## Dependencies (go.mod)
```
charmbracelet/bubbles v1.0.0
charmbracelet/bubbletea v1.3.10
charmbracelet/lipgloss v1.1.0
gopkg.in/yaml.v3 v3.0.1
```

## Build & run
```bash
rtk go build -o t9s ./cmd/main.go   # build
./t9s --talosconfig ~/.talos/config  # run
rtk go test ./...                    # all tests
```

## Current WIP (as of 2026-05-31)
Three test files added/modified (not yet committed):
- `internal/talos/parse_test.go` — NEW: tests for parseContainerLines (system/k8s/tree-marker variants), normalizeContainerStatus, parseKubeletVersionFromSpec
- `internal/ui/version_test.go` — NEW: tests for checkVersionMismatch and computeScrollStart  
- `internal/ui/render_test.go` — MODIFIED: render tests for all views (processes, disks, containers, addresses, logs, nodes, services) checking height budget, cursor visibility, width constraints

## Release
GoReleaser config in `.goreleaser.yml`:
- Builds: linux/darwin × amd64/arm64
- Artifacts: tar.gz archives, deb/rpm/apk packages
- Homebrew tap: `florianspk/homebrew-tap`
- Version injected via `-X main.version={{.Version}}`

<!-- /project-context -->
<!-- rtk-instructions v2 -->
# RTK (Rust Token Killer) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with `rtk`**. If RTK has a dedicated filter, it uses it. If not, it passes through unchanged. This means RTK is always safe to use.

**Important**: Even in command chains with `&&`, use `rtk`:
```bash
# ❌ Wrong
git add . && git commit -m "msg" && git push

# ✅ Correct
rtk git add . && rtk git commit -m "msg" && rtk git push
```

## RTK Commands by Workflow

### Build & Compile (80-90% savings)
```bash
rtk cargo build         # Cargo build output
rtk cargo check         # Cargo check output
rtk cargo clippy        # Clippy warnings grouped by file (80%)
rtk tsc                 # TypeScript errors grouped by file/code (83%)
rtk lint                # ESLint/Biome violations grouped (84%)
rtk prettier --check    # Files needing format only (70%)
rtk next build          # Next.js build with route metrics (87%)
```

### Test (60-99% savings)
```bash
rtk cargo test          # Cargo test failures only (90%)
rtk go test             # Go test failures only (90%)
rtk jest                # Jest failures only (99.5%)
rtk vitest              # Vitest failures only (99.5%)
rtk playwright test     # Playwright failures only (94%)
rtk pytest              # Python test failures only (90%)
rtk rake test           # Ruby test failures only (90%)
rtk rspec               # RSpec test failures only (60%)
rtk test <cmd>          # Generic test wrapper - failures only
```

### Git (59-80% savings)
```bash
rtk git status          # Compact status
rtk git log             # Compact log (works with all git flags)
rtk git diff            # Compact diff (80%)
rtk git show            # Compact show (80%)
rtk git add             # Ultra-compact confirmations (59%)
rtk git commit          # Ultra-compact confirmations (59%)
rtk git push            # Ultra-compact confirmations
rtk git pull            # Ultra-compact confirmations
rtk git branch          # Compact branch list
rtk git fetch           # Compact fetch
rtk git stash           # Compact stash
rtk git worktree        # Compact worktree
```

Note: Git passthrough works for ALL subcommands, even those not explicitly listed.

### GitHub (26-87% savings)
```bash
rtk gh pr view <num>    # Compact PR view (87%)
rtk gh pr checks        # Compact PR checks (79%)
rtk gh run list         # Compact workflow runs (82%)
rtk gh issue list       # Compact issue list (80%)
rtk gh api              # Compact API responses (26%)
```

### JavaScript/TypeScript Tooling (70-90% savings)
```bash
rtk pnpm list           # Compact dependency tree (70%)
rtk pnpm outdated       # Compact outdated packages (80%)
rtk pnpm install        # Compact install output (90%)
rtk npm run <script>    # Compact npm script output
rtk npx <cmd>           # Compact npx command output
rtk prisma              # Prisma without ASCII art (88%)
```

### Files & Search (60-75% savings)
```bash
rtk ls <path>           # Tree format, compact (65%)
rtk read <file>         # Code reading with filtering (60%)
rtk grep <pattern>      # Search grouped by file (75%). Format flags (-c, -l, -L, -o, -Z) run raw.
rtk find <pattern>      # Find grouped by directory (70%)
```

### Analysis & Debug (70-90% savings)
```bash
rtk err <cmd>           # Filter errors only from any command
rtk log <file>          # Deduplicated logs with counts
rtk json <file>         # JSON structure without values
rtk deps                # Dependency overview
rtk env                 # Environment variables compact
rtk summary <cmd>       # Smart summary of command output
rtk diff                # Ultra-compact diffs
```

### Infrastructure (85% savings)
```bash
rtk docker ps           # Compact container list
rtk docker images       # Compact image list
rtk docker logs <c>     # Deduplicated logs
rtk kubectl get         # Compact resource list
rtk kubectl logs        # Deduplicated pod logs
```

### Network (65-70% savings)
```bash
rtk curl <url>          # Compact HTTP responses (70%)
rtk wget <url>          # Compact download output (65%)
```

### Meta Commands
```bash
rtk gain                # View token savings statistics
rtk gain --history      # View command history with savings
rtk discover            # Analyze Claude Code sessions for missed RTK usage
rtk proxy <cmd>         # Run command without filtering (for debugging)
rtk init                # Add RTK instructions to CLAUDE.md
rtk init --global       # Add RTK to ~/.claude/CLAUDE.md
```

## Token Savings Overview

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| Package Managers | pnpm, npm, npx | 70-90% |
| Files | ls, read, grep, find | 60-75% |
| Infrastructure | docker, kubectl | 85% |
| Network | curl, wget | 65-70% |

Overall average: **60-90% token reduction** on common development operations.
<!-- /rtk-instructions -->
