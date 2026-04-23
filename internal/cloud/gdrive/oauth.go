package gdrive

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type LoopbackResult struct {
	Code  string
	State string
	Err   error
}

func GeneratePKCE() (verifier, challenge string) {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return
}

func RandomState() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func BuildAuthURL(redirectURI, state, codeChallenge string) string {
	v := url.Values{}
	v.Set("client_id", ClientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("scope", strings.Join([]string{ScopeOpenID, ScopeEmail, ScopeDriveFile}, " "))
	v.Set("access_type", "offline")
	v.Set("prompt", "consent")
	v.Set("code_challenge", codeChallenge)
	v.Set("code_challenge_method", "S256")
	v.Set("state", state)
	return AuthURL + "?" + v.Encode()
}

func StartLoopbackListener(expectedState string) (int, func(), <-chan LoopbackResult, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0") // MUST be 127.0.0.1, never 0.0.0.0
	if err != nil {
		return 0, nil, nil, err
	}
	port := ln.Addr().(*net.TCPAddr).Port

	resultCh := make(chan LoopbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		res := LoopbackResult{
			Code:  q.Get("code"),
			State: q.Get("state"),
		}
		if e := q.Get("error"); e != "" {
			res.Err = errors.New(e)
		}
		// Constant-time state check. Reject forged callbacks that
		// could come from any local process able to hit 127.0.0.1:P.
		if res.Err == nil && subtle.ConstantTimeCompare([]byte(res.State), []byte(expectedState)) != 1 {
			res.Err = errors.New("oauth: state mismatch")
		}

		status := http.StatusOK
		if res.Err != nil {
			status = http.StatusBadRequest
		}
		select {
		case resultCh <- res:
		default:
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		if res.Err != nil {
			_, _ = w.Write([]byte(`<!doctype html><html><body style="font:14px system-ui;padding:2rem">
<p>OAuth callback failed. You can close this tab and try again in TidyBill.</p></body></html>`))
			return
		}
		_, _ = w.Write([]byte(`<!doctype html><html><body style="font:14px system-ui;padding:2rem">
<p>TidyBill is connected. You can close this tab.</p></body></html>`))
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
	return port, shutdown, resultCh, nil
}

func oauthConfig(redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
		Scopes:       []string{ScopeOpenID, ScopeEmail, ScopeDriveFile},
	}
}

func ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth2.Token, error) {
	return oauthConfig(redirectURI).Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
}

func FetchUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", UserInfoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("userinfo: %s", resp.Status)
	}
	var payload struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Email, nil
}

func RevokeToken(ctx context.Context, refreshToken string) error {
	body := "token=" + url.QueryEscape(refreshToken)
	req, err := http.NewRequestWithContext(ctx, "POST", RevokeURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("revoke: %s", resp.Status)
	}
	return nil
}
