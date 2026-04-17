//go:build windows

package cmd

func terminateOracleProcessTree(rootPID int) ([]int, error) {
	_ = rootPID
	return nil, nil
}
