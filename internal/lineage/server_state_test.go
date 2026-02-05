package lineage

import (
	"os"
	"testing"
	"time"
)

func TestServerStateSaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpFile := "/tmp/test-server-state-" + time.Now().Format("20060102150405") + ".json"
	defer os.Remove(tmpFile)

	// Create test server
	server := &HetznerServer{
		ID:         12345,
		Name:       "test-server",
		IP:         "192.168.1.1",
		SSHPort:    22,
		SSHKeyID:   67890,
		UserKeyIDs: []int64{111, 222},
		Datacenter: "fsn1-dc14",
	}

	// Save state
	err := SaveServerState(tmpFile, server, "test-token")
	if err != nil {
		t.Fatalf("SaveServerState failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatalf("State file was not created")
	}

	// Load state
	state, err := LoadServerState(tmpFile)
	if err != nil {
		t.Fatalf("LoadServerState failed: %v", err)
	}

	// Verify fields
	if state.ServerID != server.ID {
		t.Errorf("ServerID mismatch: expected %d, got %d", server.ID, state.ServerID)
	}
	if state.ServerName != server.Name {
		t.Errorf("ServerName mismatch: expected %s, got %s", server.Name, state.ServerName)
	}
	if state.ServerIP != server.IP {
		t.Errorf("ServerIP mismatch: expected %s, got %s", server.IP, state.ServerIP)
	}
	if state.SSHPort != server.SSHPort {
		t.Errorf("SSHPort mismatch: expected %d, got %d", server.SSHPort, state.SSHPort)
	}
	if state.SSHKeyID != server.SSHKeyID {
		t.Errorf("SSHKeyID mismatch: expected %d, got %d", server.SSHKeyID, state.SSHKeyID)
	}
	if len(state.UserKeyIDs) != len(server.UserKeyIDs) {
		t.Errorf("UserKeyIDs length mismatch: expected %d, got %d", len(server.UserKeyIDs), len(state.UserKeyIDs))
	}
	if state.Datacenter != server.Datacenter {
		t.Errorf("Datacenter mismatch: expected %s, got %s", server.Datacenter, state.Datacenter)
	}
	if state.HetznerToken != "test-token" {
		t.Errorf("HetznerToken mismatch: expected test-token, got %s", state.HetznerToken)
	}
}

func TestRemoveServerState(t *testing.T) {
	t.Parallel()

	tmpFile := "/tmp/test-remove-state-" + time.Now().Format("20060102150405") + ".json"

	// Create test server
	server := &HetznerServer{
		ID:       12345,
		Name:     "test-server",
		IP:       "192.168.1.1",
		SSHPort:  22,
		SSHKeyID: 67890,
	}

	// Save state
	err := SaveServerState(tmpFile, server, "test-token")
	if err != nil {
		t.Fatalf("SaveServerState failed: %v", err)
	}

	// Remove state
	err = RemoveServerState(tmpFile)
	if err != nil {
		t.Fatalf("RemoveServerState failed: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Fatalf("State file should be removed")
	}

	// Removing non-existent file should not error
	err = RemoveServerState(tmpFile)
	if err != nil {
		t.Fatalf("RemoveServerState on non-existent file should not error: %v", err)
	}
}
