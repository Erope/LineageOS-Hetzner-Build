package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ServerState struct {
	ServerID    int64     `json:"server_id"`
	ServerName  string    `json:"server_name"`
	ServerIP    string    `json:"server_ip"`
	SSHPort     int       `json:"ssh_port"`
	SSHKeyID    int64     `json:"ssh_key_id"`
	UserKeyIDs  []int64   `json:"user_key_ids"`
	Datacenter  string    `json:"datacenter"`
	CreatedAt   time.Time `json:"created_at"`
	HetznerToken string   `json:"hetzner_token"`
}

func SaveServerState(filePath string, server *HetznerServer, token string) error {
	state := ServerState{
		ServerID:     server.ID,
		ServerName:   server.Name,
		ServerIP:     server.IP,
		SSHPort:      server.SSHPort,
		SSHKeyID:     server.SSHKeyID,
		UserKeyIDs:   server.UserKeyIDs,
		Datacenter:   server.Datacenter,
		CreatedAt:    time.Now(),
		HetznerToken: token,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal server state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("write server state: %w", err)
	}

	return nil
}

func LoadServerState(filePath string) (*ServerState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read server state: %w", err)
	}

	var state ServerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal server state: %w", err)
	}

	return &state, nil
}

func RemoveServerState(filePath string) error {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove server state: %w", err)
	}
	return nil
}

func CleanupServerFromState(statePath string) error {
	state, err := LoadServerState(statePath)
	if err != nil {
		return fmt.Errorf("load server state: %w", err)
	}

	client := NewHetznerClient(state.HetznerToken)
	ctx := context.Background()

	// Delete server
	if err := client.DeleteServer(ctx, state.ServerID); err != nil {
		return fmt.Errorf("delete server %d: %w", state.ServerID, err)
	}

	// Delete SSH keys
	if err := client.DeleteSSHKey(ctx, state.SSHKeyID); err != nil {
		return fmt.Errorf("delete ssh key %d: %w", state.SSHKeyID, err)
	}

	for _, keyID := range state.UserKeyIDs {
		if err := client.DeleteSSHKey(ctx, keyID); err != nil {
			return fmt.Errorf("delete user ssh key %d: %w", keyID, err)
		}
	}

	// Remove state file
	if err := RemoveServerState(statePath); err != nil {
		return err
	}

	return nil
}
