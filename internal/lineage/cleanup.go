package lineage

import (
	"context"
	"fmt"
	"log"
)

// CleanupPersistedServer attempts to cleanup a server from persisted state
func CleanupPersistedServer(ctx context.Context, cfg Config) error {
	state, err := LoadServerState(cfg.ServerStatePath)
	if err != nil {
		return fmt.Errorf("load server state: %w", err)
	}

	if state == nil {
		log.Printf("no persisted server state found at %s", cfg.ServerStatePath)
		return nil
	}

	log.Printf("found persisted server state: id=%d name=%s ip=%s", state.ServerID, state.ServerName, state.ServerIP)

	hetznerClient := NewHetznerClient(cfg.HetznerToken)

	// Check if server still exists
	exists, err := hetznerClient.ServerExists(ctx, state.ServerID)
	if err != nil {
		return fmt.Errorf("check server existence: %w", err)
	}

	if !exists {
		log.Printf("server %d no longer exists, cleaning up state file", state.ServerID)
		if err := DeleteServerState(cfg.ServerStatePath); err != nil {
			log.Printf("warning: failed to delete state file: %v", err)
		}
		return nil
	}

	// Server exists, attempt to delete it
	log.Printf("deleting server %d...", state.ServerID)
	if err := hetznerClient.DeleteServer(ctx, state.ServerID); err != nil {
		return fmt.Errorf("failed to delete server %d: %w", state.ServerID, err)
	}
	log.Printf("successfully deleted server %d", state.ServerID)

	// Delete SSH key if present
	if state.SSHKeyID != 0 {
		log.Printf("deleting SSH key %d...", state.SSHKeyID)
		if err := hetznerClient.DeleteSSHKey(ctx, state.SSHKeyID); err != nil {
			log.Printf("warning: failed to delete ssh key %d: %v", state.SSHKeyID, err)
		} else {
			log.Printf("successfully deleted SSH key %d", state.SSHKeyID)
		}
	}

	// Clean up state file
	if err := DeleteServerState(cfg.ServerStatePath); err != nil {
		log.Printf("warning: failed to delete state file: %v", err)
	} else {
		log.Printf("cleaned up state file %s", cfg.ServerStatePath)
	}

	return nil
}
