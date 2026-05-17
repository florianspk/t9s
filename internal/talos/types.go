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
