package rclone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type RC struct {
	addr, user, pass string
	http             *http.Client
}

func NewRC(addr, user, pass string) *RC {
	return &RC{addr: addr, user: user, pass: pass, http: newRCHTTPClient()}
}

// newRCHTTPClient returns the HTTP client used to talk to the local rclone rcd
// subprocess. rcd's RC endpoints like operations/copyfile are SYNCHRONOUS —
// they don't return until the underlying cloud upload/download completes.
// For Proton Drive, that includes encryption + chunked upload, easily 30+ s
// for even a small file. So:
//   - dial + TLS handshake stay short (loopback, should be instant)
//   - ResponseHeaderTimeout NOT set (rcd legitimately takes minutes)
//   - total Timeout generous (10 min) — covers any reasonable backup size
//     while still preventing runaway hangs from a stuck rcd.
// Callers should still pass a context with a per-call deadline if they want
// finer control.
func newRCHTTPClient() *http.Client {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
	}
	return &http.Client{Transport: t, Timeout: 10 * time.Minute}
}

func (c *RC) Call(ctx context.Context, method string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.addr+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.user, c.pass)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rclone rc %s: %s: %s", method, resp.Status, string(raw))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
