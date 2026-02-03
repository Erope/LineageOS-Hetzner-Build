package lineage

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHClient struct {
	Addr       string
	User       string
	PrivateKey []byte
	KnownHosts string
	Timeout    time.Duration
}

func NewSSHClient(addr, user string, privateKey []byte, knownHostsPath string, timeout time.Duration) (*SSHClient, error) {
	if len(privateKey) == 0 {
		return nil, fmt.Errorf("private key is required")
	}
	if knownHostsPath == "" {
		return nil, fmt.Errorf("known hosts file is required")
	}
	return &SSHClient{
		Addr:       addr,
		User:       user,
		PrivateKey: privateKey,
		KnownHosts: knownHostsPath,
		Timeout:    timeout,
	}, nil
}

func (c *SSHClient) Run(ctx context.Context, command string) (string, string, error) {
	client, err := c.dial()
	if err != nil {
		return "", "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return stdout.String(), stderr.String(), ctx.Err()
	case err := <-done:
		if err != nil {
			return stdout.String(), stderr.String(), fmt.Errorf("run command: %w", err)
		}
		return stdout.String(), stderr.String(), nil
	}
}

func (c *SSHClient) Upload(ctx context.Context, remotePath string, content io.Reader, mode os.FileMode) error {
	client, err := c.dial()
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	session.Stdin = content
	quotedPath := shellQuote(remotePath)
	chmodCommand := fmt.Sprintf("chmod %#o %s", mode.Perm(), quotedPath)
	writeCommand := fmt.Sprintf("cat > %s && %s", quotedPath, chmodCommand)
	command := fmt.Sprintf("sh -c %s", shellQuote(writeCommand))

	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("upload file: %w", err)
		}
		return nil
	}
}

func (c *SSHClient) Download(ctx context.Context, remotePath, localPath string) error {
	client, err := c.dial()
	if err != nil {
		return err
	}
	defer client.Close()

	file, err := os.OpenFile(filepath.Clean(localPath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stdout = file
	session.Stderr = &stderr

	command := fmt.Sprintf("cat %s", shellQuote(remotePath))
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("download file: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil
	}
}

func (c *SSHClient) dial() (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey(c.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse ssh key: %w", err)
	}

	hostKeyCallback, err := knownhosts.New(filepath.Clean(c.KnownHosts))
	if err != nil {
		return nil, fmt.Errorf("load known hosts: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            c.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.Timeout,
	}

	conn, err := net.DialTimeout("tcp", c.Addr, c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("dial tcp: %w", err)
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, c.Addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh handshake: %w", err)
	}
	return ssh.NewClient(clientConn, chans, reqs), nil
}

func GenerateEphemeralSSHKey() (privatePEM []byte, publicKey string, err error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	privateBytes, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return nil, "", fmt.Errorf("marshal private key: %w", err)
	}

	privatePEM = pem.EncodeToMemory(privateBytes)
	sshPublicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return nil, "", fmt.Errorf("marshal public key: %w", err)
	}
	publicKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey)))
	return privatePEM, publicKey, nil
}
