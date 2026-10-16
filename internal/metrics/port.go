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

	// Cache process names by PID for fast lookup
	procMap := make(map[int32]string)
	procs, _ := process.Processes()
	for _, p := range procs {
		if name, err := p.Name(); err == nil && name != "" {
			procMap[p.Pid] = name
		}
	}

	seen := make(map[string]bool)
	var result []PortInfo

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

		procName := procMap[c.Pid]
		if procName == "" {
			procName = "System / Unknown"
		}

		result = append(result, PortInfo{
			Port:        port,
			Protocol:    "TCP",
			LocalAddr:   addr,
			Status:      "LISTEN",
			PID:         c.Pid,
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
