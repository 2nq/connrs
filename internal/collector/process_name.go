package collector

import (
	"fmt"
	"path/filepath"
	"strings"
)

func normalizeProcessName(pid int32, candidates ...string) string {
	for _, candidate := range candidates {
		name := strings.TrimSpace(candidate)
		if name == "" {
			continue
		}

		base := strings.TrimSpace(filepath.Base(name))
		if base == "." || base == string(filepath.Separator) || base == "" {
			continue
		}

		return base
	}

	switch pid {
	case 0:
		return "System Idle Process"
	case 4:
		return "System"
	default:
		return fmt.Sprintf("PID %d", pid)
	}
}
