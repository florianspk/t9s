package talos

type Node struct {
	Hostname    string
	IP          string // actual node IP used for talosctl -n
	DisplayIP   string // shown in UI (may include VIP)
	Role        string
	Version     string // Talos version
	KubeVersion string // Kubernetes/kubelet version (fetched async)
	Status      string
}

type Service struct {
	ID      string
	State   string
	Healthy string
}

type Extension struct {
	Name        string
	Version     string
	Description string
}

type StatsResult struct {
	ID       string
	CPUNanos int64   // cumulative CPU nanoseconds
	MemoryMB float64 // memory in MB
}

type CatalogExtension struct {
	Name        string
	ImageRef    string // full ref with digest, e.g. ghcr.io/siderolabs/amd-ucode:v1.6.4@sha256:...
	Author      string
	Description string
}

type DiskInfo struct {
	Dev    string
	Model  string
	Serial string
	Type   string
	Size   string
}

// VolumeInfo holds filesystem-level usage for one Talos-managed volume.
type VolumeInfo struct {
	ID        string // Talos volume ID, e.g. "EPHEMERAL"
	DiskID    string // device name, e.g. "sda"
	Mount     string // mount point, e.g. "/var/mnt/ephemeral"
	FS        string // filesystem type, e.g. "ext4"
	Size      uint64 // total size in bytes
	Available uint64 // available bytes
	Phase     string // "ready", "failed", etc.
}

// --- LVM (Talos 1.14+: LVMPhysicalVolumeStatus / LVMVolumeGroupStatus / LVMLogicalVolumeStatus) ---
//
// Parsed from `talosctl get pvs|vgs|lvs` table output (stable across Talos
// versions; JSON spec keys are not). Sizes are kept as the human strings talosctl
// prints (e.g. "40 GiB").

type LVMPhysicalVolume struct {
	Device      string // e.g. /dev/sdb1
	VolumeGroup string // "" when not yet attached to a VG
	Size        string
	Free        string
}

type LVMVolumeGroup struct {
	Name string
	Size string
	Free string
	PVs  string
	LVs  string
}

type LVMLogicalVolume struct {
	Name        string // <vg>/<lv> path as printed
	VolumeGroup string
	Layout      string // linear | raid0 | raid1 | raid10
	Size        string
	Active      string // "true"/"active"/…
	DMDevice    string // kernel device-mapper node, e.g. "dm-0"
}

// --- LVM desired state (from LVMVolumeGroupConfig / LVMLogicalVolumeConfig
// documents in the machine config, reflected as *spec resources) ---

type LVMVolumeGroupConfig struct {
	Name            string
	PhysicalVolumes []string
}

type LVMLogicalVolumeConfig struct {
	Name        string
	VolumeGroup string
	Type        string // linear | raid0 | raid1 | raid10
	Size        string // absolute size, or "80%" when sized against the VG
	Mirrors     uint64
	Stripes     uint64
}

// LVMValidationError is a problem the LVM controllers could not resolve, e.g.
// a disk claimed by two volume groups.
type LVMValidationError struct {
	VolumeGroup string
	Message     string
}

type ProcessInfo struct {
	PID     string
	State   string
	CPUTime string
	ResMem  string
	Command string
}

type ContainerInfo struct {
	Namespace string
	ID        string
	Image     string
	PID       string
	Status    string
}

type AddressInfo struct {
	Interface string
	Address   string
	Family    string
	Scope     string
}
