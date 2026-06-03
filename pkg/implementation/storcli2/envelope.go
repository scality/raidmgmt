// Package storcli2 models the shared envelope of the storcli2/perccli2 JSON
// output and decodes it, surfacing the in-JSON errors these CLIs report at
// exit code 0. It holds only the parts every storcli2 getter needs: the
// command envelope and the decoder. The per-section structs (controllers,
// physical drives, virtual drives) live in their respective getter packages,
// the same way the mdadm/ssacli adapters keep their parsing structs in their
// own files.
package storcli2

import "encoding/json"

type (
	// CmdOutput is the top-level envelope returned by every storcli2 invocation.
	CmdOutput struct {
		Controllers []Controller `json:"Controllers"`
	}

	// Controller wraps the per-controller command status and its raw response
	// data. Getters decode ResponseData into their own section structs.
	Controller struct {
		CommandStatus CommandStatus   `json:"Command Status"`
		ResponseData  json.RawMessage `json:"Response Data"`
	}

	// CommandStatus reports the outcome of a command for one controller.
	// Controller is "any" because storcli2 reports it as an int (e.g. 5) in some
	// error payloads and as a string (e.g. "0") in show outputs.
	CommandStatus struct {
		CLIVersion      string           `json:"CLI Version"`
		OperatingSystem string           `json:"Operating system"`
		Controller      any              `json:"Controller,omitempty"`
		StatusCode      int              `json:"Status Code"`
		Status          string           `json:"Status"`
		Description     string           `json:"Description"`
		DetailedStatus  []DetailedStatus `json:"Detailed Status,omitempty"`
	}

	// DetailedStatus carries per-target error details. storcli2 adds an
	// "ErrType" field and identifies the target either by "VD" or by
	// "EID:Slt"/"PID" (PID may be the int persistent id or the string "-").
	DetailedStatus struct {
		VD      any    `json:"VD,omitempty"`
		EIDSlot string `json:"EID:Slt,omitempty"`
		PID     any    `json:"PID,omitempty"`
		Status  string `json:"Status"`
		ErrType string `json:"ErrType,omitempty"`
		ErrCd   int    `json:"ErrCd"`
		ErrMsg  string `json:"ErrMsg"`
	}
)
