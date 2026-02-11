package lineage

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	Addr       string
	User       string
	PrivateKey []byte
	Timeout    time.Duration
	Stdout     io.Writer // 可选的实时输出目标（如 os.Stdout）
	Stderr     io.Writer // 可选的实时错误输出目标
}

func NewSSHClient(addr, user string, privateKey []byte, timeout time.Duration) (*SSHClient, error) {
	if len(privateKey) == 0 {
		return nil, fmt.Errorf("private key is required")
	}
	return &SSHClient{
		Addr:       addr,
		User:       user,
		PrivateKey: privateKey,
		Timeout:    timeout,
	}, nil
}

func (c *SSHClient) Run(ctx context.Context, command string) (string, string, error) {
	log.Printf("%s %s", commandLogPrefix, command)
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

	// 创建带实时输出的 buffer
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	stdoutWriter := newLineWriter(stdoutBuf, c.Stdout)
	stderrWriter := newLineWriter(stderrBuf, c.Stderr)

	session.Stdout = stdoutWriter
	session.Stderr = stderrWriter

	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return stdoutBuf.String(), stderrBuf.String(), ctx.Err()
	case err := <-done:
		// 确保刷新剩余内容
		stdoutWriter.flush()
		stderrWriter.flush()
		if out := strings.TrimSpace(stdoutBuf.String()); out != "" {
			log.Printf("[SSH][stdout]\n%s", out)
		}
		if errOut := strings.TrimSpace(stderrBuf.String()); errOut != "" {
			log.Printf("[SSH][stderr]\n%s", errOut)
		}
		if err != nil {
			return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("run command: %w", err)
		}
		return stdoutBuf.String(), stderrBuf.String(), nil
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

	// Skip host key verification for ephemeral cloud servers.
	// This is safe because:
	// 1. The IP comes directly from trusted Hetzner API after server creation
	// 2. We inject our own SSH key during server creation
	// 3. Servers are ephemeral and deleted after build completion
	config := &ssh.ClientConfig{
		User:            c.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		Timeout:         c.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return c.connect(config)
}

func (c *SSHClient) connect(config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := net.DialTimeout("tcp", c.Addr, c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("dial tcp: %w", err)
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, c.Addr, config)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			return nil, fmt.Errorf("ssh handshake failed and connection close failed: %w", errors.Join(err, closeErr))
		}
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

// lineWriter 是一个 io.Writer，它在遇到换行符时实时输出到 out，同时保留所有内容到 buf
type lineWriter struct {
	buf    *bytes.Buffer
	out    io.Writer
	prefix string
}

func newLineWriter(buf *bytes.Buffer, out io.Writer) *lineWriter {
	return &lineWriter{
		buf:    buf,
		out:    out,
		prefix: "",
	}
}

func (w *lineWriter) Write(p []byte) (n int, err error) {
	// 先写入 buffer
	n, err = w.buf.Write(p)
	if err != nil {
		return n, err
	}

	// 如果没有实时输出目标，直接返回
	if w.out == nil {
		return n, nil
	}

	// 处理前缀（上次未换行的内容）
	data := w.prefix + string(p)
	w.prefix = ""

	// 按换行符分割
	lines := strings.Split(data, "\n")

	// 最后一行如果没有换行符，作为前缀保留
	if !strings.HasSuffix(data, "\n") {
		w.prefix = lines[len(lines)-1]
		lines = lines[:len(lines)-1]
	}

	// 输出完整的行
	for _, line := range lines {
		if _, err := fmt.Fprintln(w.out, line); err != nil {
			return n, err
		}
	}

	return n, nil
}

// flush 刷新剩余的前缀内容
func (w *lineWriter) flush() {
	if w.prefix != "" && w.out != nil {
		fmt.Fprint(w.out, w.prefix)
		w.prefix = ""
	}
}
