package tool

import (
	"fmt"
	"os"
	"strings"
)

func GetChildPid(pid int) (int, error) {
	path := fmt.Sprintf("/proc/%d/task/%d/children", pid, pid)
	spec, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	if len(spec) == 0 {
		return pid, nil
	}

	n, err := parsePID(string(spec))
	if err != nil {
		return 0, fmt.Errorf("can't parse %s: %v", path, err)
	}

	return n, nil
}

func parsePID(spec string) (int, error) {
	if strings.Trim(spec, "\n") == "0" {
		return 0, fmt.Errorf("no child process found")
	}

	var pid int
	n, err := fmt.Sscanf(spec, "%d", &pid)
	if n != 1 || err != nil {
		return 0, fmt.Errorf("invalid format: %s", spec)
	}

	return pid, nil
}
