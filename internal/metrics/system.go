package metrics

import (
	"sync"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// SystemStats contains system-wide metrics.
type SystemStats struct {
	CPUPercent   float64
	MemTotal     uint64
	MemUsed      uint64
	MemPercent   float64
	ProcessCount int
}

// HistoryRingBuffer stores a sliding window of historical floats (for sparklines).
type HistoryRingBuffer struct {
	data     []float64
	capacity int
	mu       sync.RWMutex
}

// NewHistoryRingBuffer creates a new ring buffer with given capacity.
func NewHistoryRingBuffer(capacity int) *HistoryRingBuffer {
	return &HistoryRingBuffer{
		data:     make([]float64, 0, capacity),
		capacity: capacity,
	}
}

// Push appends a value, dropping oldest if full.
func (rb *HistoryRingBuffer) Push(val float64) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if len(rb.data) >= rb.capacity {
		rb.data = rb.data[1:]
	}
	rb.data = append(rb.data, val)
}

// Values returns a copy of the current values.
func (rb *HistoryRingBuffer) Values() []float64 {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	res := make([]float64, len(rb.data))
	copy(res, rb.data)
	return res
}

// GetSystemStats collects CPU and Memory usage.
func GetSystemStats() (SystemStats, error) {
	stats := SystemStats{}

	// CPU percent (non-blocking instant delta with 0 duration)
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		stats.CPUPercent = cpuPercents[0]
	}

	// Virtual memory
	vMem, err := mem.VirtualMemory()
	if err == nil {
		stats.MemTotal = vMem.Total
		stats.MemUsed = vMem.Used
		stats.MemPercent = vMem.UsedPercent
	}

	return stats, nil
}
