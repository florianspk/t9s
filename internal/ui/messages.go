package ui

import (
	"time"

	"github.com/florianspk/t9s/internal/talos"
)

type tickMsg time.Time

type nodesLoadedMsg struct {
	nodes []talos.Node
	err   error
}

type servicesLoadedMsg struct {
	services []talos.Service
	err      error
}

type extensionsLoadedMsg struct {
	extensions []talos.Extension
	err        error
}

type machineConfigLoadedMsg struct {
	content string
	err     error
}

type statsLoadedMsg struct {
	stats []talos.StatsResult
	err   error
}

type catalogLoadedMsg struct {
	catalog []talos.CatalogExtension
	err     error
}

type disksLoadedMsg struct {
	disks []talos.DiskInfo
	err   error
}

type processesLoadedMsg struct {
	processes []talos.ProcessInfo
	err       error
}

type containersLoadedMsg struct {
	containers []talos.ContainerInfo
	err        error
}

type addressesLoadedMsg struct {
	addresses []talos.AddressInfo
	err       error
}

type healthLineMsg string
type healthDoneMsg struct{}

type actionDoneMsg struct {
	action string
	nodeIP string // IP of the node the action targeted
	err    error
}

type logLineMsg string
type logDoneMsg struct{}

type dmesgLineMsg string
type dmesgDoneMsg struct{}

type upgradeLineMsg string
type upgradeDoneMsg struct{ err error }

type kubeVersionLoadedMsg struct {
	version string
	err     error
}

type clientVersionMsg struct {
	version string // talosctl client version, e.g. "v1.11.0"
}

type kubeVersionsMsg struct {
	versions map[string]string // node IP → kubelet version
}

type editorDoneMsg struct{ err error }

type machineConfigAppliedMsg struct {
	err  error
	file string // temp file to clean up
}
