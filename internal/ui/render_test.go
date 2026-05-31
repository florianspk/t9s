package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/florianspk/t9s/internal/talos"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newTestApp(width, height int) App {
	return App{width: width, height: height}
}

func makeProcesses(n int) []talos.ProcessInfo {
	p := make([]talos.ProcessInfo, n)
	for i := range p {
		p[i] = talos.ProcessInfo{
			PID:     fmt.Sprintf("%d", 1000+i),
			State:   "S",
			CPUTime: "0:01",
			ResMem:  "10240",
			Command: "/usr/bin/containerd --config /etc/containerd/config.toml --root /var/lib/containerd --state /run/containerd --log-level debug",
		}
	}
	return p
}

func makeDisks() []talos.DiskInfo {
	return []talos.DiskInfo{
		// Long model name on purpose — must be truncated to colModel=20
		{Dev: "sda", Type: "HDD", Model: "Samsung 870 EVO SATA III 2.5 Inch 1TB Internal SSD", Serial: "S3Z3NX0M123456", Size: "1.0TB"},
		{Dev: "sdb", Type: "SSD", Model: "WD Blue", Serial: "WD-WX11A1234567890", Size: "500GB"},
		{Dev: "sdc", Type: "NVMe", Model: "WD Black SN850X NVMe SSD 2TB", Serial: "234567WD0123456", Size: "2.0TB"},
	}
}

func makeContainers(n int) []talos.ContainerInfo {
	c := make([]talos.ContainerInfo, n)
	for i := range c {
		c[i] = talos.ContainerInfo{
			Namespace: "k8s.io",
			ID:        fmt.Sprintf("abc%012d", i),
			Image:     "ghcr.io/siderolabs/extensions:v1.6.4-sha256-abcdef1234567890abcdef1234567890abcdef",
			PID:       fmt.Sprintf("%d", 2000+i),
			Status:    "Running",
		}
	}
	return c
}

func makeNodes(n int) []talos.Node {
	nodes := make([]talos.Node, n)
	for i := range nodes {
		nodes[i] = talos.Node{
			Hostname:    fmt.Sprintf("talos-node-%02d.example.internal", i),
			IP:          fmt.Sprintf("10.0.0.%d", i+1),
			DisplayIP:   fmt.Sprintf("10.0.0.%d", i+1),
			Role:        "worker",
			Version:     "v1.13.3",
			KubeVersion: "v1.31.0",
			Status:      "ready",
		}
	}
	if len(nodes) > 0 {
		nodes[0].Role = "controlplane"
	}
	return nodes
}

func makeServices(n int) []talos.Service {
	svcs := make([]talos.Service, n)
	for i := range svcs {
		svcs[i] = talos.Service{
			ID:      fmt.Sprintf("service-%02d", i),
			State:   "Running",
			Healthy: "OK",
		}
	}
	return svcs
}

func makeAddresses(n int) []talos.AddressInfo {
	a := make([]talos.AddressInfo, n)
	for i := range a {
		a[i] = talos.AddressInfo{
			Interface: fmt.Sprintf("eth%d", i),
			Address:   fmt.Sprintf("192.168.%d.%d/24", i, i+1),
			Family:    "inet",
			Scope:     "global very-long-scope-label-that-might-overflow-the-column",
		}
	}
	return a
}

func lineCount(s string) int { return strings.Count(s, "\n") }

func maxLineWidth(s string) int {
	max := 0
	for _, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > max {
			max = w
		}
	}
	return max
}

// ── processes ─────────────────────────────────────────────────────────────────

func TestRenderProcessesHeightBudget(t *testing.T) {
	cases := []struct {
		width, height, n, cur int
		wrap                  bool
	}{
		{80, 20, 5, 0, false},
		{80, 20, 5, 0, true},
		{80, 20, 20, 5, true},
		{80, 20, 20, 15, true},  // cursor late, many wrapped rows above
		{60, 15, 20, 10, true},  // narrow terminal
		{120, 30, 50, 25, true}, // wide terminal
	}
	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("w%d_h%d_n%d_cur%d_wrap%v", tc.width, tc.height, tc.n, tc.cur, tc.wrap), func(t *testing.T) {
			app := newTestApp(tc.width, tc.height)
			app.processes = makeProcesses(tc.n)
			app.wrapMode = tc.wrap
			app.listScroll = tc.cur
			out := app.renderProcesses(tc.height)
			if got := lineCount(out); got > tc.height {
				t.Errorf("output has %d lines, want ≤ %d\n%s", got, tc.height, out)
			}
		})
	}
}

func TestRenderProcessesCursorAlwaysVisible(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		for _, cur := range []int{0, 5, 15, 19} {
			wrap, cur := wrap, cur
			t.Run(fmt.Sprintf("wrap%v_cur%d", wrap, cur), func(t *testing.T) {
				app := newTestApp(80, 25)
				app.processes = makeProcesses(20)
				app.wrapMode = wrap
				app.listScroll = cur
				out := app.renderProcesses(22)
				if !strings.Contains(out, "▶") {
					t.Errorf("▶ cursor not visible\n%s", out)
				}
			})
		}
	}
}

// Specific regression: many wrapped items above the cursor must not push cursor off-screen.
func TestRenderProcessesCursorVisibleWhenManyWrappedRowsAbove(t *testing.T) {
	app := newTestApp(60, 20) // narrow → many wrap lines per item
	app.processes = makeProcesses(20)
	app.wrapMode = true
	app.listScroll = 19 // last item
	out := app.renderProcesses(17)
	if !strings.Contains(out, "▶") {
		t.Error("▶ cursor not visible when many wrapped items above it")
	}
	if got := lineCount(out); got > 17 {
		t.Errorf("height budget exceeded: %d lines > 17", got)
	}
}

// ── disks ─────────────────────────────────────────────────────────────────────

func TestRenderDisksNoLineExceedsWidth(t *testing.T) {
	for _, width := range []int{60, 80, 120, 200} {
		width := width
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			app := newTestApp(width, 25)
			app.disks = makeDisks()
			out := app.renderDisks(22)
			if got := maxLineWidth(out); got > width {
				t.Errorf("a line is %d chars wide, terminal is only %d\n%s", got, width, out)
			}
		})
	}
}

func TestRenderDisksCursorAlwaysVisible(t *testing.T) {
	for _, cur := range []int{0, 1, 2} {
		cur := cur
		t.Run(fmt.Sprintf("cur%d", cur), func(t *testing.T) {
			app := newTestApp(80, 25)
			app.disks = makeDisks()
			app.listScroll = cur
			out := app.renderDisks(22)
			if !strings.Contains(out, "▶") {
				t.Errorf("▶ cursor not visible at cur=%d\n%s", cur, out)
			}
		})
	}
}

// MODEL column must never exceed modelW (i.e. must always be truncated).
func TestRenderDisksModelNeverOverflows(t *testing.T) {
	for _, width := range []int{60, 80, 120} {
		width := width
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			app := newTestApp(width, 25)
			app.disks = makeDisks()
			// wrapMode ON or OFF: model must always be truncated
			for _, wrap := range []bool{false, true} {
				app.wrapMode = wrap
				out := app.renderDisks(22)
				if got := maxLineWidth(out); got > width {
					t.Errorf("wrap=%v: max line width %d > terminal width %d", wrap, got, width)
				}
			}
		})
	}
}

// ── containers ────────────────────────────────────────────────────────────────

func TestRenderContainersCursorAlwaysVisible(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		for _, cur := range []int{0, 5, 9} {
			wrap, cur := wrap, cur
			t.Run(fmt.Sprintf("wrap%v_cur%d", wrap, cur), func(t *testing.T) {
				app := newTestApp(80, 25)
				app.containers = makeContainers(10)
				app.wrapMode = wrap
				app.contCur = cur
				out := app.renderContainers(22)
				if !strings.Contains(out, "▶") {
					t.Errorf("▶ cursor not visible\n%s", out)
				}
			})
		}
	}
}

func TestRenderContainersHeightBudget(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		for _, cur := range []int{0, 5, 9} {
			wrap, cur := wrap, cur
			t.Run(fmt.Sprintf("wrap%v_cur%d", wrap, cur), func(t *testing.T) {
				app := newTestApp(80, 25)
				app.containers = makeContainers(10)
				app.wrapMode = wrap
				app.contCur = cur
				const h = 22
				out := app.renderContainers(h)
				if got := lineCount(out); got > h {
					t.Errorf("output has %d lines, want ≤ %d", got, h)
				}
			})
		}
	}
}

// ── addresses ────────────────────────────────────────────────────────────────

func TestRenderAddressesCursorAlwaysVisible(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		for _, cur := range []int{0, 3, 9} {
			wrap, cur := wrap, cur
			t.Run(fmt.Sprintf("wrap%v_cur%d", wrap, cur), func(t *testing.T) {
				app := newTestApp(80, 25)
				app.addresses = makeAddresses(10)
				app.wrapMode = wrap
				app.listScroll = cur
				out := app.renderAddresses(22)
				if !strings.Contains(out, "▶") {
					t.Errorf("▶ cursor not visible\n%s", out)
				}
			})
		}
	}
}

// ── renderLinesCursor (logs/dmesg/health) ─────────────────────────────────────

func makeLines(n int) []string {
	ls := make([]string, n)
	for i := range ls {
		ls[i] = fmt.Sprintf("line %04d: some log content that is long enough to wrap on a narrow terminal and test the budget", i)
	}
	return ls
}

func TestRenderLinesCursorHeightBudget(t *testing.T) {
	lines := makeLines(50)
	cases := []struct{ width, maxRows, cur int }{
		{80, 20, 0},
		{80, 20, 25},
		{80, 20, 49},
		{40, 10, 25}, // narrow → each line wraps → tightest budget
		{40, 10, 49},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("w%d_r%d_cur%d", tc.width, tc.maxRows, tc.cur), func(t *testing.T) {
			out := renderLinesCursor(lines, tc.cur, tc.width, tc.maxRows)
			if got := lineCount(out); got > tc.maxRows {
				t.Errorf("got %d lines, want ≤ %d", got, tc.maxRows)
			}
		})
	}
}

func TestRenderLinesCursorAlwaysVisible(t *testing.T) {
	lines := makeLines(50)
	for _, cur := range []int{0, 10, 25, 49} {
		cur := cur
		t.Run(fmt.Sprintf("cur%d", cur), func(t *testing.T) {
			out := renderLinesCursor(lines, cur, 80, 20)
			if !strings.Contains(out, "▶") {
				t.Errorf("▶ not visible at cur=%d", cur)
			}
		})
	}
}

// ── node list ─────────────────────────────────────────────────────────────────

func TestRenderNodeListNoLineExceedsWidth(t *testing.T) {
	for _, width := range []int{80, 120, 160, 220} {
		width := width
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			app := newTestApp(width, 30)
			app.nodes = makeNodes(3)
			out := app.renderNodeList(25)
			if got := maxLineWidth(out); got > width {
				t.Errorf("line width %d > terminal width %d", got, width)
			}
		})
	}
}

func TestRenderNodeListCursorAlwaysVisible(t *testing.T) {
	for _, cur := range []int{0, 1, 2} {
		cur := cur
		t.Run(fmt.Sprintf("cur%d", cur), func(t *testing.T) {
			app := newTestApp(120, 30)
			app.nodes = makeNodes(3)
			app.nodeCur = cur
			out := app.renderNodeList(25)
			if !strings.Contains(out, "▶") {
				t.Errorf("▶ not visible at cur=%d", cur)
			}
		})
	}
}

func TestRenderNodeListHeightBudget(t *testing.T) {
	cases := []struct{ n, cur, height int }{
		{3, 0, 20},
		{3, 2, 20},
		{20, 0, 20},
		{20, 10, 20},
		{20, 19, 20},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("n%d_cur%d_h%d", tc.n, tc.cur, tc.height), func(t *testing.T) {
			app := newTestApp(120, 30)
			app.nodes = makeNodes(tc.n)
			app.nodeCur = tc.cur
			out := app.renderNodeList(tc.height)
			if got := lineCount(out); got > tc.height {
				t.Errorf("output %d lines > budget %d", got, tc.height)
			}
		})
	}
}

// NAME column must be at least 20 chars even on narrow terminals.
func TestRenderNodeListResponsiveMinColHost(t *testing.T) {
	for _, width := range []int{60, 80, 120, 200} {
		width := width
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			app := newTestApp(width, 20)
			app.nodes = makeNodes(1)
			out := app.renderNodeList(15)
			if got := maxLineWidth(out); got > width {
				t.Errorf("line width %d > terminal width %d", got, width)
			}
		})
	}
}

// K8S version column must appear and not overflow.
func TestRenderNodeListK8sVersionColumn(t *testing.T) {
	app := newTestApp(160, 20)
	app.nodes = makeNodes(2)
	out := app.renderNodeList(15)
	if !strings.Contains(out, "K8S") {
		t.Error("K8S column header not found")
	}
	if !strings.Contains(out, "v1.31.0") {
		t.Error("K8S version v1.31.0 not displayed")
	}
	if got := maxLineWidth(out); got > 160 {
		t.Errorf("line width %d > terminal width 160", got)
	}
}

// ── services ──────────────────────────────────────────────────────────────────

func TestRenderServicesCursorAlwaysVisible(t *testing.T) {
	nodes := makeNodes(1)
	for _, cur := range []int{0, 3, 6} {
		cur := cur
		t.Run(fmt.Sprintf("cur%d", cur), func(t *testing.T) {
			app := newTestApp(120, 25)
			app.services = makeServices(7)
			app.svcCur = cur
			app.selNode = &nodes[0]
			out := app.renderServices(20)
			if !strings.Contains(out, "▶") {
				t.Errorf("▶ not visible at cur=%d", cur)
			}
		})
	}
}

func TestRenderServicesHeightBudget(t *testing.T) {
	nodes := makeNodes(1)
	for _, cur := range []int{0, 3, 6} {
		cur := cur
		t.Run(fmt.Sprintf("cur%d", cur), func(t *testing.T) {
			app := newTestApp(120, 25)
			app.services = makeServices(7)
			app.svcCur = cur
			app.selNode = &nodes[0]
			const h = 20
			out := app.renderServices(h)
			if got := lineCount(out); got > h {
				t.Errorf("output %d lines > budget %d", got, h)
			}
		})
	}
}

func TestRenderServicesNoLineExceedsWidth(t *testing.T) {
	nodes := makeNodes(1)
	for _, width := range []int{80, 120, 200} {
		width := width
		t.Run(fmt.Sprintf("w%d", width), func(t *testing.T) {
			app := newTestApp(width, 25)
			app.services = makeServices(7)
			app.selNode = &nodes[0]
			out := app.renderServices(20)
			if got := maxLineWidth(out); got > width {
				t.Errorf("line width %d > terminal width %d", got, width)
			}
		})
	}
}
