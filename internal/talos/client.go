package talos

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type Client struct {
	ConfigPath string
	Context    string
}

func New(configPath, ctx string) *Client {
	return &Client{ConfigPath: configPath, Context: ctx}
}

func (c *Client) baseArgs() []string {
	var args []string
	if c.ConfigPath != "" {
		args = append(args, "--talosconfig", c.ConfigPath)
	}
	if c.Context != "" {
		args = append(args, "--context", c.Context)
	}
	return args
}

// GetClientVersion returns the talosctl binary version without any network call.
func GetClientVersion(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "talosctl", "version", "--client").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Tag:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Tag:"))
		}
	}
	return ""
}

func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	cmdArgs := append(c.baseArgs(), args...)
	cmd := exec.CommandContext(ctx, "talosctl", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// parseJSONStream decodes a stream of JSON objects (pretty-printed or NDJSON).
func parseJSONStream[T any](data []byte) ([]T, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var results []T
	for {
		var item T
		err := dec.Decode(&item)
		if err == io.EOF {
			break
		}
		if err != nil {
			// skip malformed token, advance past it
			dec.Token() //nolint:errcheck
			continue
		}
		results = append(results, item)
	}
	return results, nil
}

// --- Members / Nodes ---

type memberEnvelope struct {
	Node string `json:"node"` // IP of the Talos node that produced this response
	Spec struct {
		Addresses       []string `json:"addresses"`
		Hostname        string   `json:"hostname"`
		MachineType     string   `json:"machineType"`
		OperatingSystem string   `json:"operatingSystem"`
	} `json:"spec"`
}

func (c *Client) GetNodes(ctx context.Context) ([]Node, error) {
	data, err := c.run(ctx, "get", "members", "-o", "json")
	if err != nil {
		return nil, err
	}
	envs, err := parseJSONStream[memberEnvelope](data)
	if err != nil {
		return nil, err
	}

	// Deduplicate by hostname — each cluster node reports all members.
	// Use the `node` field as the target IP: it's the actual node IP that
	// responded, not a VIP (which may appear as addresses[0] on controlplanes).
	seen := map[string]bool{}
	var nodes []Node
	for _, e := range envs {
		if seen[e.Spec.Hostname] {
			continue
		}
		seen[e.Spec.Hostname] = true

		// Find the actual node IP to use with talosctl -n.
		// The `node` field is the responding node's IP; for a member, one of
		// its addresses should match it (the self-report case).
		// For workers reported by another node, fall back to the last address
		// (Talos lists VIP first, actual IP last for multi-address nodes).
		ip := ""
		for _, addr := range e.Spec.Addresses {
			if addr == e.Node {
				ip = addr
				break
			}
		}
		if ip == "" && len(e.Spec.Addresses) > 0 {
			ip = e.Spec.Addresses[len(e.Spec.Addresses)-1]
		}

		// Display IP: node IP only (VIP would be confusing in the UI).
		displayIP := ip

		version := e.Spec.OperatingSystem
		if i := strings.Index(version, "("); i >= 0 {
			version = strings.TrimRight(version[i+1:], ")")
		}
		nodes = append(nodes, Node{
			Hostname:  e.Spec.Hostname,
			IP:        ip,        // used for talosctl -n <ip>
			DisplayIP: displayIP, // shown in the UI
			Role:      e.Spec.MachineType,
			Version:   version,
			Status:    "ready",
		})
	}
	return nodes, nil
}

// --- Services ---

func (c *Client) GetServices(ctx context.Context, node string) ([]Service, error) {
	// `talosctl get servicestatuses` is not registered; use `talosctl services`.
	// Output: NODE  SERVICE  STATE  HEALTH  LAST CHANGE  LAST EVENT
	data, err := c.run(ctx, "services", "-n", node)
	if err != nil {
		return nil, err
	}
	var services []Service
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // skip header and empty lines
		}
		fields := strings.Fields(line)
		// fields: [NODE, SERVICE, STATE, HEALTH, ...]
		if len(fields) < 4 {
			continue
		}
		services = append(services, Service{
			ID:      fields[1],
			State:   fields[2],
			Healthy: fields[3],
		})
	}
	return services, nil
}

// --- Extensions ---

type extensionEnvelope struct {
	Spec struct {
		Metadata struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		} `json:"metadata"`
	} `json:"spec"`
}

func (c *Client) GetExtensions(ctx context.Context, node string) ([]Extension, error) {
	data, err := c.run(ctx, "get", "extensions", "-n", node, "-o", "json")
	if err != nil {
		return nil, err
	}
	envs, err := parseJSONStream[extensionEnvelope](data)
	if err != nil {
		return nil, err
	}
	var exts []Extension
	for _, e := range envs {
		exts = append(exts, Extension{
			Name:        e.Spec.Metadata.Name,
			Version:     e.Spec.Metadata.Version,
			Description: e.Spec.Metadata.Description,
		})
	}
	return exts, nil
}

// --- Machine Config ---

func (c *Client) GetMachineConfig(ctx context.Context, node string) (string, error) {
	data, err := c.run(ctx, "get", "machineconfig", "-n", node, "-o", "yaml")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// --- Stats ---

func (c *Client) GetStats(ctx context.Context, node string) ([]StatsResult, error) {
	data, err := c.run(ctx, "stats", "-n", node)
	if err != nil {
		return nil, err
	}
	var results []StatsResult
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		// Output: NODE  NAMESPACE  ID  MEMORY(MB)  CPU
		if len(fields) < 5 {
			continue
		}
		memMB, _ := strconv.ParseFloat(fields[3], 64)
		cpuNanos, _ := strconv.ParseInt(fields[4], 10, 64)
		results = append(results, StatsResult{
			ID:       fields[2],
			MemoryMB: memMB,
			CPUNanos: cpuNanos,
		})
	}
	return results, nil
}

// --- Streaming ---

func (c *Client) StreamLogs(ctx context.Context, node, service string, ch chan<- string) {
	cmdArgs := append(c.baseArgs(), "logs", "-n", node, "-f", "--tail", "500", service)
	cmd := exec.CommandContext(ctx, "talosctl", cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- fmt.Sprintf("ERROR: %v", err)
		return
	}
	if err := cmd.Start(); err != nil {
		ch <- fmt.Sprintf("ERROR: %v", err)
		return
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			cmd.Process.Kill() //nolint:errcheck
			return
		case ch <- scanner.Text():
		}
	}
	cmd.Wait() //nolint:errcheck
}

func (c *Client) StreamDmesg(ctx context.Context, node string, ch chan<- string) {
	cmdArgs := append(c.baseArgs(), "dmesg", "-n", node, "-f")
	cmd := exec.CommandContext(ctx, "talosctl", cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- fmt.Sprintf("ERROR: %v", err)
		return
	}
	if err := cmd.Start(); err != nil {
		ch <- fmt.Sprintf("ERROR: %v", err)
		return
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			cmd.Process.Kill() //nolint:errcheck
			return
		case ch <- scanner.Text():
		}
	}
	cmd.Wait() //nolint:errcheck
}

func (c *Client) UpgradeTalos(ctx context.Context, node, image string, preserve bool, ch chan<- string) error {
	cmdArgs := append(c.baseArgs(), "upgrade", "-n", node, "--image", image)
	if preserve {
		cmdArgs = append(cmdArgs, "--preserve")
	}
	return c.runStreaming(ctx, ch, cmdArgs...)
}

func (c *Client) UpgradeK8s(ctx context.Context, version string, ch chan<- string) error {
	cmdArgs := append(c.baseArgs(), "upgrade-k8s", "--to", version)
	return c.runStreaming(ctx, ch, cmdArgs...)
}

// GetKubernetesVersion returns the current Kubernetes version by reading
// the kubelet image tag from KubeletSpec — available on all node types.
// Image format: "ghcr.io/siderolabs/kubelet:v1.31.0"
func (c *Client) GetKubernetesVersion(ctx context.Context, node string) (string, error) {
	data, err := c.run(ctx, "get", "kubeletspec", "-n", node, "-o", "json")
	if err != nil {
		return "", err
	}
	const prefix = "ghcr.io/siderolabs/kubelet:"
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if i := strings.Index(line, prefix); i >= 0 {
			tag := line[i+len(prefix):]
			for j, ch := range tag {
				if ch == '"' || ch == ',' || ch == ' ' {
					tag = tag[:j]
					break
				}
			}
			return strings.TrimPrefix(tag, "v"), nil
		}
	}
	return "", fmt.Errorf("kubelet image version not found in kubeletspec")
}

// GetKubeVersions fetches the kubelet version for each node concurrently.
// Returns a map of node IP → version string (e.g. "v1.31.0").
func (c *Client) GetKubeVersions(ctx context.Context, nodes []Node) map[string]string {
	type result struct {
		ip  string
		ver string
	}
	ch := make(chan result, len(nodes))
	for _, n := range nodes {
		n := n
		go func() {
			v, err := c.GetKubernetesVersion(ctx, n.IP)
			if err == nil {
				ch <- result{n.IP, "v" + v}
			} else {
				ch <- result{n.IP, ""}
			}
		}()
	}
	out := make(map[string]string, len(nodes))
	for range nodes {
		r := <-ch
		if r.ver != "" {
			out[r.ip] = r.ver
		}
	}
	return out
}

// --- Extension Catalog ---

// GetExtensionCatalog fetches the list of available Talos extensions for a given Talos version
// by pulling the ghcr.io/siderolabs/extensions:<talosVersion> image and reading descriptions.yaml.
func (c *Client) GetExtensionCatalog(ctx context.Context, talosVersion string) ([]CatalogExtension, error) {
	image := "ghcr.io/siderolabs/extensions:" + talosVersion
	cmd := exec.CommandContext(ctx, "crane", "export", "--platform", "linux/amd64", image, "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("crane not found: %w", err)
	}

	tr := tar.NewReader(stdout)
	var descData []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			cmd.Wait() //nolint:errcheck
			return nil, fmt.Errorf("read catalog: %w", err)
		}
		if hdr.Name == "descriptions.yaml" || strings.HasSuffix(hdr.Name, "/descriptions.yaml") {
			descData, err = io.ReadAll(tr)
			if err != nil {
				cmd.Wait() //nolint:errcheck
				return nil, err
			}
			break
		}
	}

	if err := cmd.Wait(); err != nil && descData == nil {
		return nil, fmt.Errorf("crane export: %s", strings.TrimSpace(stderr.String()))
	}
	if descData == nil {
		return nil, fmt.Errorf("descriptions.yaml not found for %s", talosVersion)
	}

	return parseExtensionCatalog(descData)
}

func parseExtensionCatalog(data []byte) ([]CatalogExtension, error) {
	var result []CatalogExtension
	lines := strings.Split(string(data), "\n")

	var cur *CatalogExtension
	inDesc := false

	flush := func() {
		if cur != nil {
			cur.Description = strings.TrimSpace(cur.Description)
			result = append(result, *cur)
			cur = nil
		}
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		// Top-level key: image ref (no leading whitespace, or YAML explicit key '? ...')
		if line[0] != ' ' && line[0] != '\t' && line[0] != ':' {
			flush()
			ref := strings.TrimSuffix(line, ":")
			ref = strings.TrimPrefix(ref, "? ") // YAML explicit key marker
			name := ref
			if i := strings.LastIndex(ref, "/"); i >= 0 {
				name = ref[i+1:]
			}
			if i := strings.Index(name, ":"); i >= 0 {
				name = name[:i]
			}
			cur = &CatalogExtension{Name: name, ImageRef: ref}
			inDesc = false
			continue
		}
		if cur == nil {
			continue
		}
		// YAML explicit value indicator ': field: val' — strip the leading ': '
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ": ") {
			trimmed = trimmed[2:]
		}
		if strings.HasPrefix(trimmed, "author:") {
			cur.Author = strings.TrimSpace(strings.TrimPrefix(trimmed, "author:"))
			inDesc = false
		} else if strings.HasPrefix(trimmed, "description:") {
			inDesc = true
		} else if inDesc {
			// Strip up to 4 spaces of leading indent from description continuation
			cur.Description += strings.TrimPrefix(strings.TrimPrefix(line, "    "), "  ") + " "
		}
	}
	flush()

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// --- Node actions ---

func (c *Client) Reboot(ctx context.Context, node string) error {
	_, err := c.run(ctx, "reboot", "-n", node)
	return err
}

func (c *Client) Shutdown(ctx context.Context, node string) error {
	_, err := c.run(ctx, "shutdown", "-n", node)
	return err
}

// --- Disks ---

func (c *Client) GetDisks(ctx context.Context, node string) ([]DiskInfo, error) {
	data, err := c.run(ctx, "disks", "-n", node)
	if err != nil {
		return nil, err
	}
	var result []DiskInfo
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Fields(line)
		// DEV MODEL SERIAL TYPE UUID WWID MODALIAS NAME SIZE_VAL SIZE_UNIT BUS_PATH
		if len(f) < 5 {
			continue
		}
		size := ""
		// SIZE is 2 fields before BUS_PATH (last field)
		if len(f) >= 3 {
			size = f[len(f)-3] + " " + f[len(f)-2]
		}
		result = append(result, DiskInfo{
			Dev:    f[0],
			Model:  f[1],
			Serial: f[2],
			Type:   f[3],
			Size:   size,
		})
	}
	return result, nil
}

// --- Processes ---

func (c *Client) GetProcesses(ctx context.Context, node string) ([]ProcessInfo, error) {
	data, err := c.run(ctx, "processes", "-n", node)
	if err != nil {
		return nil, err
	}
	var result []ProcessInfo
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Fields(line)
		// NODE PID PPID STATE THREADS CPU-TIME VIRTUALM RESIDENTM COMMAND...
		if len(f) < 9 {
			continue
		}
		cmd := strings.Join(f[8:], " ")
		result = append(result, ProcessInfo{
			PID:     f[1],
			State:   f[3],
			CPUTime: f[5],
			ResMem:  f[7],
			Command: cmd,
		})
	}
	return result, nil
}

// --- Containers ---

func normalizeContainerStatus(s string) string {
	switch s {
	case "CONTAINER_RUNNING":
		return "RUNNING"
	case "CONTAINER_STOPPED", "CONTAINER_EXITED":
		return "STOPPED"
	case "SANDBOX_READY":
		return "READY"
	case "SANDBOX_NOTREADY":
		return "NOT_READY"
	default:
		return s
	}
}

func parseContainerLines(data []byte) []ContainerInfo {
	var result []ContainerInfo
	for i, line := range strings.Split(string(data), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Fields(line)
		// NODE NAMESPACE [└─] ID [IMAGE] PID STATUS
		// k8s child containers have a "└─" tree marker before the ID,
		// shifting all remaining fields by one.
		// System containers may have no IMAGE field.
		if len(f) < 5 {
			continue
		}
		idIdx := 2
		if strings.HasPrefix(f[2], "└") {
			idIdx = 3
		}
		if idIdx >= len(f) {
			continue
		}
		id := f[idIdx]
		rest := f[idIdx+1:]
		var img, pid, status string
		switch len(rest) {
		case 0, 1:
			continue
		case 2:
			// No image: PID STATUS
			pid, status = rest[0], rest[1]
		default:
			// IMAGE PID STATUS
			img, pid, status = rest[0], rest[1], rest[2]
		}
		result = append(result, ContainerInfo{
			Namespace: f[1],
			ID:        id,
			Image:     img,
			PID:       pid,
			Status:    normalizeContainerStatus(status),
		})
	}
	return result
}

func (c *Client) GetContainers(ctx context.Context, node string) ([]ContainerInfo, error) {
	// Query system namespace (talos services).
	data1, err1 := c.run(ctx, "containers", "-n", node)
	// Query k8s.io namespace (kubernetes pods).
	data2, err2 := c.run(ctx, "containers", "-k", "-n", node)

	if err1 != nil && err2 != nil {
		return nil, err1
	}
	var result []ContainerInfo
	if err1 == nil {
		result = append(result, parseContainerLines(data1)...)
	}
	if err2 == nil {
		result = append(result, parseContainerLines(data2)...)
	}
	return result, nil
}

// --- Addresses ---

type addressEnvelope struct {
	Metadata struct {
		ID string `json:"id"`
	} `json:"metadata"`
	Spec struct {
		Address  string `json:"address"`
		LinkName string `json:"linkName"`
		Family   string `json:"family"`
		Scope    string `json:"scope"`
	} `json:"spec"`
}

func (c *Client) GetAddresses(ctx context.Context, node string) ([]AddressInfo, error) {
	data, err := c.run(ctx, "get", "addresses", "-n", node, "-o", "json")
	if err != nil {
		return nil, err
	}
	envs, err := parseJSONStream[addressEnvelope](data)
	if err != nil {
		return nil, err
	}
	var result []AddressInfo
	for _, e := range envs {
		iface := e.Spec.LinkName
		if iface == "" {
			// fall back: parse "eth0/10.0.0.1/24" from ID
			if parts := strings.SplitN(e.Metadata.ID, "/", 2); len(parts) > 0 {
				iface = parts[0]
			}
		}
		result = append(result, AddressInfo{
			Interface: iface,
			Address:   e.Spec.Address,
			Family:    e.Spec.Family,
			Scope:     e.Spec.Scope,
		})
	}
	return result, nil
}

// --- Health ---

func (c *Client) StreamHealth(ctx context.Context, ch chan<- string) {
	cmdArgs := append(c.baseArgs(), "health")
	cmd := exec.CommandContext(ctx, "talosctl", cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- fmt.Sprintf("ERROR: %v", err)
		return
	}
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		ch <- fmt.Sprintf("ERROR: %v", err)
		return
	}
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return
			case ch <- sc.Text():
			}
		}
	}()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			cmd.Process.Kill() //nolint:errcheck
			return
		case ch <- scanner.Text():
		}
	}
	cmd.Wait() //nolint:errcheck
}

func (c *Client) runStreaming(ctx context.Context, ch chan<- string, args ...string) error {
	cmd := exec.CommandContext(ctx, "talosctl", args...)
	outR, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	errR, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanAll := func(r interface{ Read([]byte) (int, error) }) {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return
			case ch <- sc.Text():
			}
		}
	}
	go scanAll(errR)
	scanAll(outR)
	return cmd.Wait()
}
