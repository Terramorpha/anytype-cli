package tools

import (
	"encoding/json"
	"fmt"
)

// emit prints v as indented JSON — the uniform output format for all tools
// commands, so they compose (pipe through jq into the next command).
func emit(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// ok builds a standard success result for action commands. Extra key/value
// pairs (id, blockId, …) are merged in.
func ok(extra map[string]any) map[string]any {
	m := map[string]any{"ok": true}
	for k, v := range extra {
		m[k] = v
	}
	return m
}
