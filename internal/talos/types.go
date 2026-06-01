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
	ID        string
	CPUNanos  int64   // cumulative CPU nanoseconds
	MemoryMB  float64 // memory in MB
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
