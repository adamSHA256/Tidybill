package rclone

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	bin string

	mu      sync.Mutex
	cmd     *exec.Cmd
	port    int
	rcUser  string
	rcPass  string
	cfgPath string // disposable tempfile; deleted in Stop()
}

func NewManager(bin string) *Manager {
	return &Manager{bin: bin, rcUser: "tidybill"}
}

func (m *Manager) EnsureRunning(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil && isRunning(m.cmd.Process.Pid) {
		return nil
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	m.rcPass = hex.EncodeToString(buf)

	// Point rclone at a disposable empty config file. On Unix /dev/null
	// would work, but rclone on Windows has historically misbehaved with
	// NUL as a config path (it opens the file with O_CREATE and tries
	// to write back). We use a per-process tempfile instead, cleaned up
	// in Stop(). It's created empty so rclone starts with zero remotes.
	cfgFile, err := os.CreateTemp("", "tidybill-rclone-*.conf")
	if err != nil {
		return err
	}
	_ = cfgFile.Close()
	m.cfgPath = cfgFile.Name()

	args := []string{
		"rcd",
		"--rc-addr", "127.0.0.1:0",
		"--rc-user", m.rcUser,
		"--rc-pass", m.rcPass,
		"--config", m.cfgPath,
		"--log-level", "INFO",
	}
	// Use context.Background() — NOT the caller's request context — so that
	// rclone stays alive for the lifetime of the Manager. A request context
	// is cancelled when the HTTP handler returns (or the client disconnects),
	// which would kill rclone mid-session and leave subsequent requests with
	// "connection refused". Lifecycle is managed explicitly via Stop().
	cmd := exec.CommandContext(context.Background(), m.bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd

	portCh := make(chan int, 1)
	go scanForPort(stdout, portCh, "stdout")
	go scanForPort(stderr, portCh, "stderr")

	select {
	case p := <-portCh:
		m.port = p
		return nil
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		return errors.New("rclone: rcd did not announce a port within 10s")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return ctx.Err()
	}
}

var portRx = regexp.MustCompile(`Serving remote control on http://127\.0\.0\.1:(\d+)/`)

func scanForPort(r io.Reader, ch chan<- int, tag string) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		log.Printf("rclone(%s): %s", tag, line)
		if m := portRx.FindStringSubmatch(line); m != nil {
			var p int
			fmt.Sscanf(m[1], "%d", &p)
			select {
			case ch <- p:
			default:
			}
		}
	}
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	_ = m.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() { _ = m.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = m.cmd.Process.Kill()
		<-done
	}
	m.cmd = nil
	m.port = 0
	if m.cfgPath != "" {
		_ = os.Remove(m.cfgPath)
		m.cfgPath = ""
	}
	return nil
}

func (m *Manager) Endpoint() (addr, user, pass string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("http://127.0.0.1:%d", m.port), m.rcUser, m.rcPass
}

func isRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}
