package lineage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerState represents the persisted state of a Hetzner server
type ServerState struct {
	ServerID     int64   `json:"server_id"`
	ServerName   string  `json:"server_name"`
	ServerIP     string  `json:"server_ip"`
	SSHKeyID     int64   `json:"ssh_key_id"`
	GitHubKeyIDs []int64 `json:"github_key_ids,omitempty"`
	SSHPort      int     `json:"ssh_port"`
	Datacenter   string  `json:"datacenter"`
}

// SaveServerState persists server information to a file
func SaveServerState(path string, server *HetznerServer) error {
	state := ServerState{
		ServerID:     server.ID,
		ServerName:   server.Name,
		ServerIP:     server.IP,
		SSHKeyID:     server.SSHKeyID,
		GitHubKeyIDs: server.GitHubKeyIDs,
		SSHPort:      server.SSHPort,
		Datacenter:   server.Datacenter,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal server state: %w", err)
	}

	if err := os.WriteFile(filepath.Clean(path), data, 0600); err != nil {
		return fmt.Errorf("write server state: %w", err)
	}

	return nil
}

// LoadServerState reads server information from a file
func LoadServerState(path string) (*ServerState, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read server state: %w", err)
	}

	var state ServerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal server state: %w", err)
	}

	return &state, nil
}

// DeleteServerState removes the server state file
func DeleteServerState(path string) error {
	err := os.Remove(filepath.Clean(path))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete server state: %w", err)
	}
	return nil
}
