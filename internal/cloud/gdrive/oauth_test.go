package gdrive

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestBuildAuthURL(t *testing.T) {
	verifier, challenge := GeneratePKCE()
	state := RandomState()
	redirectURI := "http://127.0.0.1:8765/oauth/callback"

	rawURL := BuildAuthURL(redirectURI, state, challenge)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}

	q := parsed.Query()

	tests := []struct {
		param string
		want  string
	}{
		{"response_type", "code"},
		{"access_type", "offline"},
		{"prompt", "consent"},
		{"code_challenge_method", "S256"},
		{"redirect_uri", redirectURI},
		{"state", state},
		{"code_challenge", challenge},
	}
	for _, tt := range tests {
		if got := q.Get(tt.param); got != tt.want {
			t.Errorf("param %q = %q, want %q", tt.param, got, tt.want)
		}
	}

	// scope must contain all three required scopes
	scope := q.Get("scope")
	for _, s := range []string{ScopeOpenID, ScopeEmail, ScopeDriveFile} {
		found := false
		for _, part := range splitScope(scope) {
			if part == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("scope %q missing from %q", s, scope)
		}
	}

	_ = verifier // used via challenge
}

func splitScope(scope string) []string {
	var out []string
	start := 0
	for i, ch := range scope {
		if ch == ' ' || ch == '+' {
			if i > start {
				out = append(out, scope[start:i])
			}
			start = i + 1
		}
	}
	if start < len(scope) {
		out = append(out, scope[start:])
	}
	return out
}

func TestLoopbackListenerSuccess(t *testing.T) {
	state := RandomState()
	port, shutdown, resultCh, err := StartLoopbackListener(state)
	if err != nil {
		t.Fatalf("StartLoopbackListener: %v", err)
	}
	defer shutdown()

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback?code=testcode&state=%s",
		port, url.QueryEscape(state))

	go func() {
		resp, err := http.Get(callbackURL) //nolint:gosec
		if err != nil {
			return
		}
		resp.Body.Close()
	}()

	res := <-resultCh
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Code != "testcode" {
		t.Fatalf("code = %q, want %q", res.Code, "testcode")
	}
	if res.State != state {
		t.Fatalf("state = %q, want %q", res.State, state)
	}
}

func TestLoopbackListenerStateMismatch(t *testing.T) {
	state := RandomState()
	port, shutdown, resultCh, err := StartLoopbackListener(state)
	if err != nil {
		t.Fatalf("StartLoopbackListener: %v", err)
	}
	defer shutdown()

	// Send a forged state
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback?code=evilcode&state=forged",
		port)

	go func() {
		resp, err := http.Get(callbackURL) //nolint:gosec
		if err != nil {
			return
		}
		resp.Body.Close()
	}()

	res := <-resultCh
	if res.Err == nil {
		t.Fatal("expected error for state mismatch, got nil")
	}
}
