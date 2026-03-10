package tor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/net/proxy"
)

const (
	torWinURL   = "https://dist.torproject.org/torbrowser/15.0.7/tor-expert-bundle-windows-x86_64-15.0.7.tar.gz"
	torLinuxURL = "https://dist.torproject.org/torbrowser/15.0.7/tor-expert-bundle-linux-x86_64-15.0.7.tar.gz"
)

// Manager handles the lifecycle of the Tor process.
type Manager struct {
	cmd        *exec.Cmd
	binaryPath string
}

// EnsureTorBinary checks for the tor binary or downloads it if missing.
func (m *Manager) EnsureTorBinary() (string, error) {
	// 1. Check if 'tor' is in PATH
	binName := "tor"
	if runtime.GOOS == "windows" {
		binName = "tor.exe"
	}

	path, err := exec.LookPath(binName)
	if err == nil {
		m.binaryPath = path
		return path, nil
	}

	// 2. Check in local bin/ folder
	localBinDir := "bin"
	localPath := filepath.Join(localBinDir, binName)
	if runtime.GOOS == "windows" {
		// In Windows bundle, it's often in tor/tor.exe inside the extracted folder
		localPath = filepath.Join(localBinDir, "tor", "tor.exe")
	} else {
		localPath = filepath.Join(localBinDir, "tor", "tor")
	}

	if _, err := os.Stat(localPath); err == nil {
		m.binaryPath = localPath
		return localPath, nil
	}

	// 3. Download if not found
	fmt.Printf("Tor binary not found. Downloading expert bundle for %s...\n", runtime.GOOS)
	url := torLinuxURL
	if runtime.GOOS == "windows" {
		url = torWinURL
	}

	if err := os.MkdirAll(localBinDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin dir: %w", err)
	}

	if err := m.downloadAndExtract(url, localBinDir); err != nil {
		return "", fmt.Errorf("failed to download/extract tor: %w", err)
	}

	m.binaryPath = localPath
	return localPath, nil
}

func (m *Manager) downloadAndExtract(url, destDir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

// Start launches a local Tor process.
func (m *Manager) Start() error {
	if m.binaryPath == "" {
		if _, err := m.EnsureTorBinary(); err != nil {
			return err
		}
	}

	// We use a temporary data directory for Tor
	dataDir, err := os.MkdirTemp("", "darker-tor-*")
	if err != nil {
		return fmt.Errorf("failed to create temp data dir: %w", err)
	}

	m.cmd = exec.Command(m.binaryPath,
		"--DataDirectory", dataDir,
		"--SocksPort", "9050",
		"--ControlPort", "9051",
		"--CookieAuthentication", "1",
		"--Log", "notice stdout",
	)

	// In a real app we might want to capture output to check for "Bootstrapped 100%"
	// For now we just start it and let CheckConnection handle readiness
	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tor: %w", err)
	}

	return nil
}

// Stop terminates the Tor process.
func (m *Manager) Stop() error {
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Kill()
	}
	return nil
}

// Client is a wrapper around http.Client that routes traffic through Tor.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Tor client using the provided SOCKS5 proxy address.
// Default Tor proxy address is typically "127.0.0.1:9050".
func NewClient(proxyAddr string) (*Client, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	dialContext := func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.Dial(network, address)
	}

	transport := &http.Transport{
		DialContext: dialContext,
		// Onion services can be slow, so we set reasonable timeouts.
		TLSHandshakeTimeout: 30 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
	}

	return &Client{
		httpClient: httpClient,
	}, nil
}

// CheckConnection verifies if the Tor proxy is reachable and functional.
func (c *Client) CheckConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to reach a known reliable onion service or just check the proxy
	// For simplicity, we just try to fetch a small page or check if we can dial
	req, err := http.NewRequestWithContext(ctx, "GET", "http://check.torproject.org", nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("tor proxy unreachable or connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status from check: %d", resp.StatusCode)
	}

	return nil
}

// Get performs an HTTP GET request through the Tor proxy.
func (c *Client) Get(url string) (*http.Response, error) {
	return c.httpClient.Get(url)
}

// Do performs an HTTP request through the Tor proxy.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}
