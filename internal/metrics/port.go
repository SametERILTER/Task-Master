package metrics

import (
	"fmt"
	"sort"
	"strings"

	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// PortInfo represents an open/listening network port and its owning process.
type PortInfo struct {
	Port        uint32
	Protocol    string
	LocalAddr   string
	Status      string
	PID         int32
	ProcessName string
}

// GetOpenPorts returns active listening ports, optionally filtered by a specific port (0 returns all).
func GetOpenPorts(targetPort uint32) ([]PortInfo, error) {
	conns, err := gopsnet.Connections("tcp")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve network connections: %w", err)
	}

	// 1. Identify listening connections and unique PIDs needed
	type listenEntry struct {
		port uint32
		addr string
		pid  int32
	}
	var entries []listenEntry
	neededPids := make(map[int32]struct{})
	seen := make(map[string]bool)

	for _, c := range conns {
		// Only consider listening ports
		if !strings.EqualFold(c.Status, "LISTEN") {
			continue
		}

		port := c.Laddr.Port
		if targetPort > 0 && port != targetPort {
			continue
		}

		addr := c.Laddr.IP
		if addr == "" {
			addr = "0.0.0.0"
		}

		key := fmt.Sprintf("%d-%d-%s", port, c.Pid, addr)
		if seen[key] {
			continue
		}
		seen[key] = true

		entries = append(entries, listenEntry{
			port: port,
			addr: addr,
			pid:  c.Pid,
		})
		if c.Pid > 0 && c.Pid != 4 {
			neededPids[c.Pid] = struct{}{}
		}
	}

	// 2. Resolve process names only for unique listening PIDs (O(K) lookup with early exit)
	procMap := make(map[int32]string, len(neededPids))
	if len(neededPids) > 0 {
		procs, _ := process.Processes()
		for _, p := range procs {
			if _, needed := neededPids[p.Pid]; needed {
				if name, err := p.Name(); err == nil && name != "" {
					procMap[p.Pid] = name
				}
				if len(procMap) == len(neededPids) {
					break
				}
			}
		}
	}

	// 3. Assemble PortInfo result
	result := make([]PortInfo, 0, len(entries))
	for _, e := range entries {
		procName := procMap[e.pid]
		if procName == "" {
			if e.pid == 4 || e.pid == 0 {
				procName = "System"
			} else {
				procName = "System / Unknown"
			}
		}

		result = append(result, PortInfo{
			Port:        e.port,
			Protocol:    "TCP",
			LocalAddr:   e.addr,
			Status:      "LISTEN",
			PID:         e.pid,
			ProcessName: procName,
		})
	}

	// Sort ports ascending
	sort.Slice(result, func(i, j int) bool {
		if result[i].Port == result[j].Port {
			return result[i].PID < result[j].PID
		}
		return result[i].Port < result[j].Port
	})

	return result, nil
}
