package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// outputOK writes a JSON success envelope to stdout:
//
//	{"ok":true,"result":<result>}
//
// This matches the shell's json_ok() function format for playbook compatibility.
func outputOK(result interface{}) {
	resultJSON, _ := json.Marshal(result)
	fmt.Fprintf(stdout, "{\"ok\":true,\"result\":%s}\n", string(resultJSON))
}

// outputError writes a JSON error envelope to stderr:
//
//	{"ok":false,"error":"<message>","code":<code>}
//
// This matches the shell's json_err() function format for playbook compatibility.
func outputError(code int, message string, details interface{}) {
	msgJSON, _ := json.Marshal(message)
	fmt.Fprintf(stderr, "{\"ok\":false,\"error\":%s,\"code\":%d}\n", string(msgJSON), code)
}

// outputErrorMessage is a convenience wrapper for outputError with code 1.
func outputErrorMessage(message string) {
	outputError(1, message, nil)
}

// mustGetString retrieves a required string flag, calling outputError and
// exiting if the flag is missing or empty.
func mustGetString(cmd *cobra.Command, flag string) string {
	val, err := cmd.Flags().GetString(flag)
	if err != nil {
		outputError(1, fmt.Sprintf("missing flag --%s", flag), nil)
		return ""
	}
	if val == "" {
		outputError(1, fmt.Sprintf("flag --%s is required", flag), nil)
		return ""
	}
	return val
}

// mustGetInt retrieves a required int flag, calling outputError and
// exiting if the flag is missing.
func mustGetInt(cmd *cobra.Command, flag string) int {
	val, err := cmd.Flags().GetInt(flag)
	if err != nil {
		outputError(1, fmt.Sprintf("missing flag --%s", flag), nil)
		return 0
	}
	return val
}
