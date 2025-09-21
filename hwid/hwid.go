package hwid

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Info struct {
	Hostname     string
	MachineID    string
	CPU          string
	MacAddresses []string
}

func Collect() (Info, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return Info{}, err
	}

	machineID := readFirstExisting(
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	)

	cpu := runtime.GOARCH + "-" + runtime.GOOS
	if runtime.GOOS == "linux" {
		if val := readCPUModel(); val != "" {
			cpu = val
		}
	}

	macs := collectMACs()

	return Info{
		Hostname:     hostname,
		MachineID:    machineID,
		CPU:          cpu,
		MacAddresses: macs,
	}, nil
}

func (i Info) HardwareID() string {
	builder := strings.Builder{}
	builder.WriteString(strings.ToUpper(strings.TrimSpace(i.Hostname)))
	builder.WriteString("|")
	builder.WriteString(strings.ToUpper(strings.TrimSpace(i.MachineID)))
	builder.WriteString("|")
	builder.WriteString(strings.ToUpper(strings.TrimSpace(i.CPU)))
	builder.WriteString("|")
	builder.WriteString(strings.Join(i.MacAddresses, ","))
	return builder.String()
}

func readFirstExisting(paths ...string) string {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}

func readCPUModel() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.ToLower(line), "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func collectMACs() []string {
	var macs []string
	interfaces, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return macs
	}
	for _, iface := range interfaces {
		name := iface.Name()
		if strings.HasPrefix(name, "lo") || strings.Contains(name, "virtual") {
			continue
		}
		addrPath := filepath.Join("/sys/class/net", name, "address")
		data, err := os.ReadFile(addrPath)
		if err != nil {
			continue
		}
		mac := strings.TrimSpace(string(data))
		if mac == "00:00:00:00:00:00" || mac == "" {
			continue
		}
		macs = append(macs, strings.ToUpper(mac))
	}
	return macs
}

func (i Info) String() string {
	return fmt.Sprintf("hostname=%s machine=%s cpu=%s macs=%v", i.Hostname, i.MachineID, i.CPU, i.MacAddresses)
}
