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
	PPID          int32
	Name          string
	CPUPercent    float64
	MemoryPercent float32
	MemoryBytes   uint64
	Username      string
	TreePrefix    string
	Depth         int
	ChildCount    int
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
		ppid, _ := p.Ppid()

		result = append(result, ProcessInfo{
			PID:         p.Pid,
			PPID:        ppid,
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

type treeNode struct {
	proc     ProcessInfo
	children []*treeNode
	matches  bool
}

// GetProcessTree retrieves running processes organized hierarchically by parent-child relationships.
func GetProcessTree(sortBy string, filter string) ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve processes: %w", err)
	}

	filterLower := strings.ToLower(strings.TrimSpace(filter))
	nodeMap := make(map[int32]*treeNode, len(procs))
	var allNodes []*treeNode

	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}

		ppid, _ := p.Ppid()
		cpuPct, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		var memBytes uint64
		if memInfo != nil {
			memBytes = memInfo.RSS
		}

		matches := true
		if filterLower != "" {
			pidStr := fmt.Sprintf("%d", p.Pid)
			matches = strings.Contains(strings.ToLower(name), filterLower) || strings.Contains(pidStr, filterLower)
		}

		node := &treeNode{
			proc: ProcessInfo{
				PID:         p.Pid,
				PPID:        ppid,
				Name:        name,
				CPUPercent:  cpuPct,
				MemoryBytes: memBytes,
			},
			matches: matches,
		}
		nodeMap[p.Pid] = node
		allNodes = append(allNodes, node)
	}

	// Link parent-children relationships and identify root nodes
	var roots []*treeNode
	for _, node := range allNodes {
		parent, hasParent := nodeMap[node.proc.PPID]
		// Prevent cycles or self-parenting
		if hasParent && parent.proc.PID != node.proc.PID {
			parent.children = append(parent.children, node)
		} else {
			roots = append(roots, node)
		}
	}

	// Filter pruning: If filter is active, only retain subtrees that contain matching nodes
	if filterLower != "" {
		var prune func(n *treeNode) bool
		prune = func(n *treeNode) bool {
			keep := n.matches
			var filteredChildren []*treeNode
			for _, ch := range n.children {
				if prune(ch) {
					filteredChildren = append(filteredChildren, ch)
					keep = true
				}
			}
			n.children = filteredChildren
			return keep
		}

		var filteredRoots []*treeNode
		for _, r := range roots {
			if prune(r) {
				filteredRoots = append(filteredRoots, r)
			}
		}
		roots = filteredRoots
	}

	// Sort nodes helper
	sortNodes := func(nodes []*treeNode) {
		switch strings.ToLower(sortBy) {
		case "mem", "memory":
			sort.Slice(nodes, func(i, j int) bool {
				if nodes[i].proc.MemoryBytes != nodes[j].proc.MemoryBytes {
					return nodes[i].proc.MemoryBytes > nodes[j].proc.MemoryBytes
				}
				return nodes[i].proc.PID < nodes[j].proc.PID
			})
		case "name":
			sort.Slice(nodes, func(i, j int) bool {
				if strings.ToLower(nodes[i].proc.Name) != strings.ToLower(nodes[j].proc.Name) {
					return strings.ToLower(nodes[i].proc.Name) < strings.ToLower(nodes[j].proc.Name)
				}
				return nodes[i].proc.PID < nodes[j].proc.PID
			})
		case "pid":
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].proc.PID < nodes[j].proc.PID
			})
		case "cpu":
			fallthrough
		default:
			sort.Slice(nodes, func(i, j int) bool {
				if nodes[i].proc.CPUPercent != nodes[j].proc.CPUPercent {
					return nodes[i].proc.CPUPercent > nodes[j].proc.CPUPercent
				}
				if nodes[i].proc.MemoryBytes != nodes[j].proc.MemoryBytes {
					return nodes[i].proc.MemoryBytes > nodes[j].proc.MemoryBytes
				}
				return nodes[i].proc.PID < nodes[j].proc.PID
			})
		}
	}

	// Sort roots
	sortNodes(roots)

	// Flatten tree into result slice using DFS
	result := make([]ProcessInfo, 0, len(allNodes))
	visited := make(map[int32]bool)

	var dfs func(n *treeNode, prefix string, isLast bool, isRoot bool, depth int)
	dfs = func(n *treeNode, prefix string, isLast bool, isRoot bool, depth int) {
		if visited[n.proc.PID] {
			return
		}
		visited[n.proc.PID] = true

		linePrefix := ""
		childContinuation := prefix
		if !isRoot {
			if isLast {
				linePrefix = prefix + "└── "
				childContinuation = prefix + "    "
			} else {
				linePrefix = prefix + "├── "
				childContinuation = prefix + "│   "
			}
		}

		pInfo := n.proc
		pInfo.TreePrefix = linePrefix
		pInfo.Depth = depth
		pInfo.ChildCount = len(n.children)
		result = append(result, pInfo)

		// Sort children
		sortNodes(n.children)

		for i, child := range n.children {
			dfs(child, childContinuation, i == len(n.children)-1, false, depth+1)
		}
	}

	for _, root := range roots {
		dfs(root, "", true, true, 0)
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
