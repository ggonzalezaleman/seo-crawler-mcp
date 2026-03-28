package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchesForceRender(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ForceRenderPatterns = []string{"/app/*", "/dashboard"}

	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com/app/settings", true},
		{"https://example.com/app/profile", true},
		{"https://example.com/dashboard", true},
		{"https://example.com/about", false},
		{"https://example.com/", false},
		{"https://example.com/app", false}, // /app/* requires /app/ + something
	}

	for _, tt := range tests {
		got := cfg.MatchesForceRender(tt.url)
		if got != tt.want {
			t.Errorf("MatchesForceRender(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestMatchesForceRender_Empty(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MatchesForceRender("https://example.com/anything") {
		t.Error("empty ForceRenderPatterns should match nothing")
	}
}

func TestForceRenderPatterns_TOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `force_render_patterns = ["/app/*", "/spa/*"]`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if len(cfg.ForceRenderPatterns) != 2 {
		t.Fatalf("ForceRenderPatterns len = %d, want 2", len(cfg.ForceRenderPatterns))
	}
	if cfg.ForceRenderPatterns[0] != "/app/*" {
		t.Errorf("ForceRenderPatterns[0] = %q, want /app/*", cfg.ForceRenderPatterns[0])
	}
}

func TestForceRenderPatterns_EnvVar(t *testing.T) {
	t.Setenv("SEO_CRAWLER_FORCE_RENDER_PATTERNS", "/spa/*,/react/*")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if len(cfg.ForceRenderPatterns) != 2 {
		t.Fatalf("ForceRenderPatterns len = %d, want 2", len(cfg.ForceRenderPatterns))
	}
	if cfg.ForceRenderPatterns[0] != "/spa/*" {
		t.Errorf("ForceRenderPatterns[0] = %q, want /spa/*", cfg.ForceRenderPatterns[0])
	}
}

func TestRobotsUnreachablePolicy_TOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `robots_unreachable_policy = "disallow"`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing config file: %v", err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if cfg.RobotsUnreachablePolicy != RobotsUnreachablePolicyDisallow {
		t.Errorf("RobotsUnreachablePolicy = %q, want disallow", cfg.RobotsUnreachablePolicy)
	}
}

func TestRobotsUnreachablePolicy_EnvVar(t *testing.T) {
	t.Setenv("SEO_CRAWLER_ROBOTS_UNREACHABLE_POLICY", "cache_then_allow")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.RobotsUnreachablePolicy != RobotsUnreachablePolicyCacheThenAllow {
		t.Errorf("RobotsUnreachablePolicy = %q, want cache_then_allow", cfg.RobotsUnreachablePolicy)
	}
}
