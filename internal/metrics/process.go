package metrics

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

// ProcessInfo holds snapshot metrics for a single operating system process.
type ProcessInfo struct {
	PID           int32
	Name          string
	CPUPercent    float64
	MemoryPercent float32
	MemoryBytes   uint64
	Username      string
}

// GetProcesses retrieves all running processes, applying filter and sorting.
func GetProcesses(sortBy string, filter string) ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve processes: %w", err)
	}

	filterLower := strings.ToLower(strings.TrimSpace(filter))
	result := make([]ProcessInfo, 0, len(procs))

	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}

		// Filter match by name or PID
		if filterLower != "" {
			pidStr := fmt.Sprintf("%d", p.Pid)
			if !strings.Contains(strings.ToLower(name), filterLower) && !strings.Contains(pidStr, filterLower) {
				continue
			}
		}

		cpuPct, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		var memBytes uint64
		if memInfo != nil {
			memBytes = memInfo.RSS
		}

		result = append(result, ProcessInfo{
			PID:         p.Pid,
			Name:        name,
			CPUPercent:  cpuPct,
			MemoryBytes: memBytes,
		})
	}

	// Sort results
	switch strings.ToLower(sortBy) {
	case "mem", "memory":
		sort.Slice(result, func(i, j int) bool {
			return result[i].MemoryBytes > result[j].MemoryBytes
		})
	case "name":
		sort.Slice(result, func(i, j int) bool {
			return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
		})
	case "pid":
		sort.Slice(result, func(i, j int) bool {
			return result[i].PID < result[j].PID
		})
	case "cpu":
		fallthrough
	default:
		sort.Slice(result, func(i, j int) bool {
			return result[i].CPUPercent > result[j].CPUPercent
		})
	}

	return result, nil
}

// ProcessDetails holds comprehensive information about a single process.
type ProcessDetails struct {
	PID         int32
	Name        string
	CPUPercent  float64
	MemoryBytes uint64
	PPID        int32
	Exe         string
	Cmdline     string
	NumThreads  int32
	CreateTime  int64
}

// GetProcessDetails fetches full details for a given PID.
func GetProcessDetails(pid int32) (*ProcessDetails, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("process PID %d not found: %w", pid, err)
	}

	name, _ := p.Name()
	cpuPct, _ := p.CPUPercent()
	memInfo, _ := p.MemoryInfo()
	var memBytes uint64
	if memInfo != nil {
		memBytes = memInfo.RSS
	}
	ppid, _ := p.Ppid()
	exe, _ := p.Exe()
	cmdline, _ := p.Cmdline()
	threads, _ := p.NumThreads()
	created, _ := p.CreateTime()

	return &ProcessDetails{
		PID:         pid,
		Name:        name,
		CPUPercent:  cpuPct,
		MemoryBytes: memBytes,
		PPID:        ppid,
		Exe:         exe,
		Cmdline:     cmdline,
		NumThreads:  threads,
		CreateTime:  created,
	}, nil
}

// KillProcess attempts to terminate a process by PID.
func KillProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("process with PID %d not found: %w", pid, err)
	}
	if err := p.Kill(); err != nil {
		return fmt.Errorf("failed to terminate PID %d (check administrator permissions): %w", pid, err)
	}
	return nil
}
