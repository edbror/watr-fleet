package adapter

import (
	"os/exec"
	"strings"

	"github.com/edbror/watr-fleet/internal/fleet"
)

// processTree is a one-shot snapshot of the system's processes, used to
// identify agents that run under an interpreter (node, python, bun) —
// where tmux's pane_current_command only says "node".
type processTree struct {
	children map[string][]string // ppid -> pids
	args     map[string]string   // pid -> full command line
}

// loadProcessTree captures pid/ppid/args for every process in one call.
// Errors degrade to an empty tree: discovery falls back to process names.
func loadProcessTree() processTree {
	tree := processTree{children: map[string][]string{}, args: map[string]string{}}
	out, err := exec.Command("ps", "-eo", "pid=,ppid=,args=").Output()
	if err != nil {
		return tree
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, ppid := fields[0], fields[1]
		tree.children[ppid] = append(tree.children[ppid], pid)
		tree.args[pid] = strings.Join(fields[2:], " ")
	}
	return tree
}

// agentUnder searches the pane's process subtree (breadth-first, bounded)
// for a command line that resolves to a known agent.
func (t processTree) agentUnder(panePID string) fleet.Agent {
	queue := []string{panePID}
	for visited := 0; len(queue) > 0 && visited < 64; visited++ {
		pid := queue[0]
		queue = queue[1:]
		if agent := fleet.AgentFromArgs(t.args[pid]); agent != fleet.AgentUnknown {
			return agent
		}
		queue = append(queue, t.children[pid]...)
	}
	return fleet.AgentUnknown
}
