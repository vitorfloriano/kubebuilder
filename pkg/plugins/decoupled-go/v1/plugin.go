/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

const (
	// PluginKey is the fully-qualified plugin identifier used in the PROJECT file layout field.
	PluginKey = "decoupled-go.example.com/v1"

	// PluginVersion is the current version of the decoupled-go plugin.
	PluginVersion = "v1.0.0-alpha"
)

// Run is the main entry point for the decoupled-go/v1 external plugin.
// It reads a PluginRequest from in, dispatches to the appropriate handler,
// and writes a PluginResponse to out.
//
// Callers (typically cmd/main.go in the plugin binary) should pass
// os.Stdin and os.Stdout:
//
//	if err := v1.Run(os.Stdin, os.Stdout); err != nil {
//	    fmt.Fprintln(os.Stderr, err)
//	    os.Exit(1)
//	}
func Run(in io.Reader, out io.Writer) error {
	var req external.PluginRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		// write a best-effort error response before returning
		writeErrorResponse(out, req, fmt.Sprintf("decode plugin request: %v", err))
		return fmt.Errorf("decode plugin request: %w", err)
	}

	resp := external.PluginResponse{
		APIVersion: req.APIVersion,
		Command:    req.Command,
		Universe:   cloneUniverse(req.Universe),
	}

	var handlerErr error
	switch req.Command {
	case "init":
		handlerErr = handleInit(&req, &resp)
	case "create api":
		handlerErr = handleCreateAPI(&req, &resp)
	case "create webhook":
		handlerErr = handleCreateWebhook(&req, &resp)
	case "edit":
		handlerErr = handleEdit(&req, &resp)
	case "flags":
		handlerErr = handleFlags(&req, &resp)
	case "metadata":
		handlerErr = handleMetadata(&req, &resp)
	default:
		// Unknown commands: pass universe through unchanged (chaining-safe).
		fmt.Fprintf(os.Stderr, "[decoupled-go/v1] unknown command %q — passing through\n", req.Command)
	}

	if handlerErr != nil {
		resp.Error = true
		resp.ErrorMsgs = []string{handlerErr.Error()}
		// Clear any partial universe changes on error.
		resp.Universe = cloneUniverse(req.Universe)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("encode plugin response: %w", err)
	}

	if _, err = io.Copy(out, bytes.NewReader(b)); err != nil {
		return fmt.Errorf("write plugin response: %w", err)
	}

	return nil
}

// cloneUniverse makes a shallow copy of the universe map so handlers can
// safely mutate resp.Universe without affecting req.Universe.
func cloneUniverse(u map[string]string) map[string]string {
	if u == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(u))
	for k, v := range u {
		out[k] = v
	}
	return out
}

// writeErrorResponse writes a JSON error response to out as a best-effort
// operation. Errors from writing are silently ignored.
func writeErrorResponse(out io.Writer, req external.PluginRequest, msg string) {
	resp := external.PluginResponse{
		APIVersion: req.APIVersion,
		Command:    req.Command,
		Universe:   req.Universe,
		Error:      true,
		ErrorMsgs:  []string{msg},
	}
	b, _ := json.Marshal(resp)
	_, _ = out.Write(b)
}

// flagValue scans args for the value following the named flag.
// Returns defaultVal if the flag is not present.
//
// Example: flagValue([]string{"--domain", "my.org"}, "--domain", "example.com")
// returns "my.org".
func flagValue(args []string, flag, defaultVal string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return defaultVal
}

// flagBool returns true if flag is present in args, false otherwise.
func flagBool(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || arg == flag+"=true" {
			return true
		}
	}
	return false
}
