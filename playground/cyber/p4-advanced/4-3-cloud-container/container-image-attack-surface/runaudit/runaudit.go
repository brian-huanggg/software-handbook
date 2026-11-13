// Package runaudit reads the JSON printed by `docker inspect` and reports
// run-time settings that hand the container more power than a service needs.
package runaudit

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
)

// Issue is one risky setting on one container.
type Issue struct {
	Container string
	Severity  string // "critical", "high" or "medium"
	Rule      string
	Detail    string
}

type inspect struct {
	Name   string
	Config struct{ User string }

	HostConfig struct {
		Privileged     bool
		CapAdd         []string
		CapDrop        []string
		PidMode        string
		NetworkMode    string
		ReadonlyRootfs bool
		SecurityOpt    []string
	}
	Mounts []struct{ Source, Destination string }
}

// dangerousCaps are capabilities that, once added, let a process do things
// the container boundary was supposed to forbid.
var dangerousCaps = map[string]string{
	"SYS_ADMIN":       "mount, namespace and cgroup control: close to root",
	"SYS_PTRACE":      "attach to other processes, including host processes under --pid=host",
	"SYS_MODULE":      "load kernel modules",
	"DAC_READ_SEARCH": "bypass file read permission checks",
	"NET_ADMIN":       "change interfaces, routes and firewall rules",
}

var sensitiveHostPaths = []string{"/", "/etc", "/proc", "/sys", "/root", "/var/lib/docker"}

// Audit decodes the array printed by `docker inspect` and checks each container.
func Audit(r io.Reader) ([]Issue, error) {
	var containers []inspect
	if err := json.NewDecoder(r).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decode docker inspect output: %w", err)
	}
	var issues []Issue
	for _, c := range containers {
		issues = append(issues, check(c)...)
	}
	return issues, nil
}

func check(c inspect) []Issue {
	name := strings.TrimPrefix(c.Name, "/")
	var out []Issue
	add := func(sev, rule, detail string) {
		out = append(out, Issue{Container: name, Severity: sev, Rule: rule, Detail: detail})
	}
	h := c.HostConfig

	if h.Privileged {
		add("critical", "privileged", "all capabilities, all host devices, no seccomp or AppArmor profile")
	}
	for _, m := range c.Mounts {
		// A read-only mount (:ro) does not help: connect() never writes to the socket file.
		if path.Base(m.Source) == "docker.sock" {
			add("critical", "docker-sock", "anyone inside can ask the daemon to start a container that mounts the host: "+m.Source)
		} else if slices.Contains(sensitiveHostPaths, path.Clean(m.Source)) {
			add("high", "sensitive-host-path", "host directory mounted into the container: "+m.Source)
		}
	}
	if !h.Privileged { // privileged already implies every capability
		for _, cap := range h.CapAdd {
			cap = strings.TrimPrefix(strings.ToUpper(cap), "CAP_")
			if why, bad := dangerousCaps[cap]; bad {
				add("high", "dangerous-capability", cap+": "+why)
			}
		}
		if !slices.ContainsFunc(h.CapDrop, func(s string) bool { return strings.EqualFold(s, "ALL") }) {
			add("medium", "capabilities-not-dropped", "the default set of 14 capabilities is still granted")
		}
	}
	if h.PidMode == "host" {
		add("high", "host-pid", "the container sees and can signal every host process")
	}
	if h.NetworkMode == "host" {
		add("medium", "host-network", "the container shares the host network stack")
	}
	if !slices.Contains(h.SecurityOpt, "no-new-privileges") && !slices.Contains(h.SecurityOpt, "no-new-privileges:true") {
		add("medium", "new-privileges-allowed", "a setuid binary inside can still gain privileges")
	}
	if u := c.Config.User; u == "" || u == "0" || strings.HasPrefix(u, "root") || strings.HasPrefix(u, "0:") {
		add("medium", "runs-as-root", "the main process runs as uid 0 inside the container")
	}
	if !h.ReadonlyRootfs {
		add("medium", "writable-rootfs", "an attacker can drop tools and persist inside the container filesystem")
	}
	return out
}
