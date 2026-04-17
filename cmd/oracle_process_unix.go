//go:build !windows

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func terminateOracleProcessTree(rootPID int) ([]int, error) {
	if rootPID <= 0 || rootPID == os.Getpid() {
		return nil, nil
	}

	pids, err := oracleProcessTree(rootPID)
	if err != nil {
		return nil, err
	}
	if len(pids) == 0 {
		return nil, nil
	}

	var killed []int
	var failures []string
	for _, pid := range pids {
		if pid <= 0 || pid == os.Getpid() {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err == nil {
			killed = append(killed, pid)
		} else if err != syscall.ESRCH {
			failures = append(failures, fmt.Sprintf("SIGTERM %d: %v", pid, err))
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		remaining := false
		for _, pid := range pids {
			if oracleProcessExists(pid) {
				remaining = true
				break
			}
		}
		if !remaining {
			return killed, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	for _, pid := range pids {
		if pid <= 0 || pid == os.Getpid() || !oracleProcessExists(pid) {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGKILL); err == nil {
			if !containsOracleIteration(killed, pid) {
				killed = append(killed, pid)
			}
		} else if err != syscall.ESRCH {
			failures = append(failures, fmt.Sprintf("SIGKILL %d: %v", pid, err))
		}
	}

	if len(failures) > 0 {
		return killed, errors.New(strings.Join(failures, "; "))
	}
	return killed, nil
}

func oracleProcessTree(rootPID int) ([]int, error) {
	parents, err := oracleProcessTable()
	if err != nil {
		return nil, err
	}
	if _, ok := parents[rootPID]; !ok && !oracleProcessExists(rootPID) {
		return nil, nil
	}

	var ordered []int
	var visit func(int)
	visit = func(pid int) {
		children := parents[pid]
		sort.Ints(children)
		for _, child := range children {
			visit(child)
		}
		ordered = append(ordered, pid)
	}
	visit(rootPID)
	return ordered, nil
}

func oracleProcessTable() (map[int][]int, error) {
	cmd := exec.Command("ps", "-axo", "pid=,ppid=")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read process table: %w", err)
	}

	parents := map[int][]int{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		pid, errPID := strconv.Atoi(fields[0])
		ppid, errPPID := strconv.Atoi(fields[1])
		if errPID != nil || errPPID != nil {
			continue
		}
		parents[ppid] = append(parents[ppid], pid)
	}
	return parents, nil
}

func oracleProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}

	cmd := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	status := strings.TrimSpace(string(output))
	if status == "" {
		return false
	}
	return !strings.HasPrefix(status, "Z")
}
