package lineage

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	hcloud "github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type HetznerClient struct {
	client *hcloud.Client
}

type HetznerServer struct {
	ID         int64
	Name       string
	IP         string
	SSHUser    string
	SSHKey     []byte
	SSHPort    int
	SSHKeyID   int64
	UserKeyIDs []int64
	Datacenter string
}

func NewHetznerClient(token string) *HetznerClient {
	return &HetznerClient{client: hcloud.NewClient(hcloud.WithToken(token))}
}

func (hc *HetznerClient) CreateServer(ctx context.Context, cfg Config) (*HetznerServer, error) {
	serverType, _, err := hc.client.ServerType.GetByName(ctx, cfg.ServerType)
	if err != nil {
		return nil, fmt.Errorf("get server type: %w", err)
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type %q not found", cfg.ServerType)
	}
	image, _, err := hc.client.Image.GetByName(ctx, cfg.ServerImage)
	if err != nil {
		return nil, fmt.Errorf("get server image: %w", err)
	}
	if image == nil {
		return nil, fmt.Errorf("server image %q not found", cfg.ServerImage)
	}

	var location *hcloud.Location
	if cfg.ServerLocation != "" {
		location, _, err = hc.client.Location.GetByName(ctx, cfg.ServerLocation)
		if err != nil {
			return nil, fmt.Errorf("get server location: %w", err)
		}
		if location == nil {
			return nil, fmt.Errorf("server location %q not found", cfg.ServerLocation)
		}
	}

	privateKey, publicKey, err := GenerateEphemeralSSHKey()
	if err != nil {
		return nil, err
	}

	sshKeyName := fmt.Sprintf("lineage-builder-%d", time.Now().Unix())
	createdKey, _, err := hc.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      sshKeyName,
		PublicKey: publicKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create ssh key: %w", err)
	}

	// Collect all SSH keys to add to the server
	sshKeys := []*hcloud.SSHKey{createdKey}
	var userKeyIDs []int64

	// Add user SSH keys if provided
	for i, userKey := range cfg.UserSSHKeys {
		userKeyName := fmt.Sprintf("lineage-builder-user-%d-%d", time.Now().Unix(), i)
		userSSHKey, _, err := hc.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
			Name:      userKeyName,
			PublicKey: userKey,
		})
		if err != nil {
			// Log warning but continue - user keys are optional
			fmt.Printf("warning: failed to add user SSH key %d: %v\n", i, err)
			continue
		}
		sshKeys = append(sshKeys, userSSHKey)
		userKeyIDs = append(userKeyIDs, userSSHKey.ID)
	}

	userData, err := readUserData(cfg.ServerUserDataPath)
	if err != nil {
		return nil, err
	}

	request := hcloud.ServerCreateOpts{
		Name:       cfg.ServerName,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		UserData:   userData,
		SSHKeys:    sshKeys,
	}

	result, _, err := hc.client.Server.Create(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}
	if result.Server == nil {
		return nil, fmt.Errorf("create server returned nil server")
	}
	server := result.Server
	if server.PublicNet.IPv4.IsUnspecified() {
		return nil, fmt.Errorf("server has no public IPv4")
	}
	ip := server.PublicNet.IPv4.IP.String()
	if ip == "" {
		return nil, fmt.Errorf("server has no public IPv4")
	}

	return &HetznerServer{
		ID:         server.ID,
		Name:       server.Name,
		IP:         ip,
		SSHUser:    "root",
		SSHKey:     privateKey,
		SSHPort:    cfg.SSHPort,
		SSHKeyID:   createdKey.ID,
		UserKeyIDs: userKeyIDs,
		Datacenter: server.Datacenter.Name,
	}, nil
}

func (hc *HetznerClient) DeleteServer(ctx context.Context, id int64) error {
	_, err := hc.client.Server.Delete(ctx, &hcloud.Server{ID: id})
	if err != nil {
		return fmt.Errorf("delete server: %w", err)
	}
	return nil
}

func (hc *HetznerClient) DeleteSSHKey(ctx context.Context, id int64) error {
	if id == 0 {
		return nil
	}
	_, err := hc.client.SSHKey.Delete(ctx, &hcloud.SSHKey{ID: id})
	if err != nil {
		return fmt.Errorf("delete ssh key: %w", err)
	}
	return nil
}

func (hc *HetznerClient) WaitForServer(ctx context.Context, serverID int64) error {
	for {
		server, _, err := hc.client.Server.GetByID(ctx, serverID)
		if err != nil {
			return fmt.Errorf("get server status: %w", err)
		}
		if server == nil {
			return fmt.Errorf("server %d not found", serverID)
		}
		if server.Status == hcloud.ServerStatusRunning {
			return nil
		}
		if err := sleepWithContext(ctx, 5*time.Second); err != nil {
			return err
		}
	}
}

func readUserData(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("read user data: %w", err)
	}
	return string(data), nil
}

func waitForPort(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	dialTimeout := func(addr string, timeout time.Duration) (netConn, error) {
		return (&net.Dialer{Timeout: timeout}).Dial("tcp", addr)
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		conn, err := dialTimeout(addr, 3*time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s", addr)
		}
		if err := sleepWithContext(ctx, 3*time.Second); err != nil {
			return err
		}
	}
}

type netConn interface {
	Close() error
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
