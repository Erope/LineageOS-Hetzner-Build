package lineage

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	hcloud "github.com/hetznercloud/hcloud-go/v2/hcloud"
	"golang.org/x/crypto/ssh"
)

type HetznerClient struct {
	client *hcloud.Client
}

type HetznerServer struct {
	ID                 int64
	Name               string
	IP                 string
	SSHUser            string
	SSHKey             []byte
	SSHPort            int
	SSHKeyID           int64
	GitHubKeyIDs       []int64
	GitHubKeyIDsReused []int64 // Keys that were found and reused, not created
	Datacenter         string
}

func NewHetznerClient(token string) *HetznerClient {
	return &HetznerClient{client: hcloud.NewClient(hcloud.WithToken(token))}
}

// computeSSHKeyFingerprint computes the MD5 fingerprint of an SSH public key
func computeSSHKeyFingerprint(publicKey string) (string, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("parse public key: %w", err)
	}

	hash := md5.Sum(pubKey.Marshal())
	fingerprint := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x:%02x",
		hash[0], hash[1], hash[2], hash[3], hash[4], hash[5], hash[6], hash[7],
		hash[8], hash[9], hash[10], hash[11], hash[12], hash[13], hash[14], hash[15])
	return fingerprint, nil
}

// findOrCreateSSHKey tries to create an SSH key, or finds and returns an existing one if it already exists
func (hc *HetznerClient) findOrCreateSSHKey(ctx context.Context, name, publicKey string) (*hcloud.SSHKey, bool, error) {
	// Try to create the key first
	createdKey, _, err := hc.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      name,
		PublicKey: publicKey,
	})
	if err == nil {
		return createdKey, false, nil
	}

	// Check if it's a uniqueness error
	if hcloud.IsError(err, hcloud.ErrorCodeUniquenessError) {
		// Key already exists, try to find it by fingerprint
		fingerprint, fpErr := computeSSHKeyFingerprint(publicKey)
		if fpErr != nil {
			return nil, false, fmt.Errorf("compute fingerprint: %w", fpErr)
		}

		existingKey, _, getErr := hc.client.SSHKey.GetByFingerprint(ctx, fingerprint)
		if getErr != nil {
			return nil, false, fmt.Errorf("get existing key by fingerprint: %w", getErr)
		}
		if existingKey == nil {
			return nil, false, fmt.Errorf("key exists but could not be found by fingerprint")
		}

		log.Printf("SSH key already exists in Hetzner (fingerprint: %s), reusing it", fingerprint)
		return existingKey, true, nil
	}

	// Other error
	return nil, false, fmt.Errorf("create ssh key: %w", err)
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

	// Collect all SSH keys to inject
	sshKeys := []*hcloud.SSHKey{createdKey}
	var githubKeyIDs []int64
	var githubKeyIDsReused []int64

	// Try to fetch and inject GitHub user SSH keys if in GitHub Actions
	githubKeys, err := GetGitHubActorSSHKeys(ctx)
	if err != nil {
		log.Printf("warning: %v", err)
	} else if len(githubKeys) > 0 {
		log.Printf("found %d SSH key(s) from GitHub user, injecting into server for debugging", len(githubKeys))
		for i, key := range githubKeys {
			// Note: Hetzner enforces uniqueness of SSH keys based on the public key
			// content, not on the key name. This timestamp-based name provides a
			// human-friendly identifier but does not prevent collisions when the
			// same public key content already exists. In such cases, findOrCreateSSHKey
			// will detect the existing key by fingerprint and reuse it.
			ghKeyName := fmt.Sprintf("github-user-key-%d-%d", time.Now().Unix(), i)
			ghKey, reused, err := hc.findOrCreateSSHKey(ctx, ghKeyName, key)
			if err != nil {
				log.Printf("warning: failed to add GitHub SSH key %d: %v", i, err)
				continue
			}
			sshKeys = append(sshKeys, ghKey)
			if reused {
				githubKeyIDsReused = append(githubKeyIDsReused, ghKey.ID)
			} else {
				githubKeyIDs = append(githubKeyIDs, ghKey.ID)
			}
		}
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
		ID:                 server.ID,
		Name:               server.Name,
		IP:                 ip,
		SSHUser:            "root",
		SSHKey:             privateKey,
		SSHPort:            cfg.SSHPort,
		SSHKeyID:           createdKey.ID,
		GitHubKeyIDs:       githubKeyIDs,
		GitHubKeyIDsReused: githubKeyIDsReused,
		Datacenter:         server.Datacenter.Name,
	}, nil
}

func (hc *HetznerClient) DeleteServer(ctx context.Context, id int64) error {
	_, err := hc.client.Server.Delete(ctx, &hcloud.Server{ID: id})
	if err != nil {
		return fmt.Errorf("delete server: %w", err)
	}
	return nil
}

func (hc *HetznerClient) ServerExists(ctx context.Context, id int64) (bool, error) {
	server, _, err := hc.client.Server.GetByID(ctx, id)
	if err != nil {
		// Check if it's a "not found" error
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check server existence: %w", err)
	}
	return server != nil, nil
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
