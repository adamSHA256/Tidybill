package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adamSHA256/tidybill/internal/config"
	"github.com/adamSHA256/tidybill/internal/database/repository"
)

const (
	githubReleasesURL = "https://api.github.com/repos/adamSHA256/tidybill/releases/latest"
	checkCooldown     = 24 * time.Hour
	httpTimeout       = 3 * time.Second
	settingKey        = "check_updates"
	lastCheckKey      = "update.last_check"
	cachedResultKey   = "update.cached_result"
)

// Result holds the outcome of an update check.
type Result struct {
	Available     bool   `json:"available"`
	CurrentVer    string `json:"current_version"`
	LatestVer     string `json:"latest_version"`
	ReleaseURL    string `json:"release_url"`
	ReleaseNotes  string `json:"release_notes"`
	PublishedAt   string `json:"published_at"`
	CheckedAt     string `json:"checked_at"`
}

// Checker performs async update checks against GitHub releases.
type Checker struct {
	settings *repository.SettingsRepository
	mu       sync.RWMutex
	cached   *Result
}

// NewChecker creates a new update checker.
func NewChecker(settings *repository.SettingsRepository) *Checker {
	c := &Checker{settings: settings}
	// Restore cached result from DB
	if raw, _ := settings.Get(cachedResultKey); raw != "" {
		var r Result
		if json.Unmarshal([]byte(raw), &r) == nil {
			c.cached = &r
		}
	}
	return c
}

// StartAutoCheck runs the initial auto-check in a goroutine if enabled and cooldown has elapsed.
func (c *Checker) StartAutoCheck() {
	go func() {
		enabled, _ := c.settings.Get(settingKey)
		if enabled != "true" {
			return
		}
		if !c.cooldownElapsed() {
			return
		}
		c.doCheck() // errors silently ignored
	}()
}

// Check performs a check. If force is true, ignores cooldown and enabled setting.
func (c *Checker) Check(force bool) (*Result, error) {
	if !force {
		enabled, _ := c.settings.Get(settingKey)
		if enabled != "true" {
			return nil, fmt.Errorf("update check is disabled")
		}
		if !c.cooldownElapsed() {
			c.mu.RLock()
			r := c.cached
			c.mu.RUnlock()
			if r != nil {
				return r, nil
			}
		}
	}
	return c.doCheck()
}

// GetCached returns the last cached result (may be nil).
func (c *Checker) GetCached() *Result {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cached
}

func (c *Checker) cooldownElapsed() bool {
	raw, _ := c.settings.Get(lastCheckKey)
	if raw == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return true
	}
	return time.Since(last) >= checkCooldown
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

func (c *Checker) doCheck() (*Result, error) {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequest("GET", githubReleasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := config.Version

	result := &Result{
		Available:    compareSemver(latest, current) > 0,
		CurrentVer:   current,
		LatestVer:    latest,
		ReleaseURL:   release.HTMLURL,
		ReleaseNotes: release.Body,
		PublishedAt:  release.PublishedAt,
		CheckedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	// Cache result
	c.mu.Lock()
	c.cached = result
	c.mu.Unlock()

	// Persist
	now := time.Now().UTC().Format(time.RFC3339)
	_ = c.settings.Set(lastCheckKey, now)
	if raw, err := json.Marshal(result); err == nil {
		_ = c.settings.Set(cachedResultKey, string(raw))
	}

	return result, nil
}

// compareSemver returns >0 if a > b, 0 if equal, <0 if a < b.
func compareSemver(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] - pb[i]
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		// Strip any pre-release suffix (e.g. "1-beta")
		clean := strings.SplitN(parts[i], "-", 2)[0]
		result[i], _ = strconv.Atoi(clean)
	}
	return result
}
