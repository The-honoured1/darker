package tor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

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
