package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ParsesExposeList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "magi.yaml")
	yaml := `
runtime:
  ollama_url: http://localhost:11434
consensus:
  max_rounds: 3
brains:
  melchior:
    model: melchior
    grpc_addr: "127.0.0.1:50051"
tools:
  expose:
    - notify_user
    - log_decision
  allowlist:
    - name: notify_user
      mode: auto
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	want := []string{"notify_user", "log_decision"}
	if len(cfg.Tools.Expose) != len(want) {
		t.Fatalf("expose = %v, want %v", cfg.Tools.Expose, want)
	}
	for i, name := range want {
		if cfg.Tools.Expose[i] != name {
			t.Fatalf("expose[%d] = %q, want %q", i, cfg.Tools.Expose[i], name)
		}
	}
}
