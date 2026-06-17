package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// HardwareDevice represents a single compute device available on the host.
type HardwareDevice struct {
	ID        string `json:"id"`
	Type      string `json:"type"`       // "cpu" or "nvidia"
	Name      string `json:"name"`
	Cores     int    `json:"cores,omitempty"`      // CPU cores
	VRAMMB    int    `json:"vram_mb,omitempty"`    // GPU VRAM in MB
	Available bool   `json:"available"`
}

// HardwareInfo is the response from GET /v1/hardware.
type HardwareInfo struct {
	Hardware []HardwareDevice `json:"hardware"`
	Active   string           `json:"active"`
}

// activeHardware tracks which hardware is currently being used for inference.
var activeHardware = "cpu"

// HardwareHandler returns all available compute hardware.
func HardwareHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices := detectHardware()

		info := HardwareInfo{
			Hardware: devices,
			Active:   activeHardware,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

// detectHardware probes the system for available compute devices.
func detectHardware() []HardwareDevice {
	var devices []HardwareDevice

	// CPU is always available
	cpu := detectCPU()
	devices = append(devices, cpu)

	// Try to detect NVIDIA GPU(s)
	gpus := detectNVIDIA()
	devices = append(devices, gpus...)

	return devices
}

// detectCPU reads CPU information from the system.
func detectCPU() HardwareDevice {
	name := "Unknown CPU"
	cores := runtime.NumCPU()

	switch runtime.GOOS {
	case "linux":
		name = readCPUInfoLinux()
	case "darwin":
		name = readCPUInfoDarwin()
	}

	return HardwareDevice{
		ID:        "cpu",
		Type:      "cpu",
		Name:      name,
		Cores:     cores,
		Available: true,
	}
}

// readCPUInfoLinux reads the CPU model name from /proc/cpuinfo.
func readCPUInfoLinux() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Linux CPU"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Linux CPU"
}

// readCPUInfoDarwin reads the CPU brand string via sysctl on macOS.
func readCPUInfoDarwin() string {
	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err != nil {
		return "Apple CPU"
	}
	return strings.TrimSpace(string(out))
}

// detectNVIDIA runs nvidia-smi to find available NVIDIA GPUs.
func detectNVIDIA() []HardwareDevice {
	out, err := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		// nvidia-smi not available — no NVIDIA GPU
		return nil
	}

	var gpus []HardwareDevice
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, line := range lines {
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		vramStr := strings.TrimSpace(parts[1])
		vram, _ := strconv.Atoi(vramStr)

		gpus = append(gpus, HardwareDevice{
			ID:        fmt.Sprintf("gpu-%d", i),
			Type:      "nvidia",
			Name:      name,
			VRAMMB:    vram,
			Available: true,
		})
	}

	return gpus
}

// ExtractHardwareChoice reads the hardware preference from the request.
// It checks the X-Hardware header first, then the request body "hardware" field.
// Returns empty string if no preference is specified.
func ExtractHardwareChoice(r *http.Request) string {
	// Check header first
	if hw := r.Header.Get("X-Hardware"); hw != "" {
		return strings.ToLower(strings.TrimSpace(hw))
	}
	return ""
}

// SetActiveHardware updates the active hardware. Returns an error if the
// requested hardware ID doesn't match any detected device.
func SetActiveHardware(id string) error {
	devices := detectHardware()
	for _, d := range devices {
		if d.ID == id {
			activeHardware = id
			return nil
		}
	}
	return fmt.Errorf("hardware %q not found", id)
}
