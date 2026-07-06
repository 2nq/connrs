package collector

import (
	"context"
	"sync"

	gopsprocess "github.com/shirou/gopsutil/v3/process"
)

type processResolver struct {
	mu    sync.Mutex
	names map[int32]string
}

func newProcessResolver() *processResolver {
	return &processResolver{
		names: map[int32]string{
			0: "System Idle Process",
			4: "System",
		},
	}
}

func (r *processResolver) nameForPID(_ context.Context, pid int32) string {
	r.mu.Lock()
	if name, ok := r.names[pid]; ok {
		r.mu.Unlock()
		return name
	}
	r.mu.Unlock()

	if pid <= 0 {
		return normalizeProcessName(pid)
	}

	proc, err := gopsprocess.NewProcess(pid)
	if err != nil {
		return normalizeProcessName(pid)
	}

	candidates := make([]string, 0, 2)

	name, err := proc.Name()
	if err == nil {
		candidates = append(candidates, name)
	}

	exe, err := proc.Exe()
	if err == nil {
		candidates = append(candidates, exe)
	}

	name = normalizeProcessName(pid, candidates...)

	r.mu.Lock()
	r.names[pid] = name
	r.mu.Unlock()
	return name
}

// prune drops cached names for PIDs no longer seen. Windows reuses PIDs, so a
// stale entry would label a brand-new process with the old process's name.
func (r *processResolver) prune(active map[int32]struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for pid := range r.names {
		if pid == 0 || pid == 4 {
			continue // fixed kernel pseudo-processes, seeded at construction
		}
		if _, ok := active[pid]; !ok {
			delete(r.names, pid)
		}
	}
}
