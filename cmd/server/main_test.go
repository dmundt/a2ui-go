package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDirAndIsDir(t *testing.T) {
	path, err := resolveDir("..", "..", "renderer", "templates")
	if err != nil {
		t.Fatalf("resolveDir existing path failed: %v", err)
	}
	if !isDir(path) {
		t.Fatalf("resolved path should be a directory: %s", path)
	}

	if _, err := resolveDir("definitely", "missing", "directory"); err == nil {
		t.Fatalf("resolveDir should fail for missing directory")
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "f.txt")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write tmp file: %v", err)
	}
	if !isDir(tmpDir) {
		t.Fatalf("expected directory true")
	}
	if isDir(tmpFile) {
		t.Fatalf("expected file false")
	}
	if isDir(filepath.Join(tmpDir, "missing")) {
		t.Fatalf("expected missing path false")
	}
}

func TestShouldStartMCPStdio(t *testing.T) {
	oldArgs := os.Args
	oldEnv, hadEnv := os.LookupEnv("ENABLE_MCP_STDIO")
	defer func() {
		os.Args = oldArgs
		if hadEnv {
			_ = os.Setenv("ENABLE_MCP_STDIO", oldEnv)
		} else {
			_ = os.Unsetenv("ENABLE_MCP_STDIO")
		}
	}()

	_ = os.Setenv("ENABLE_MCP_STDIO", "1")
	os.Args = []string{"server"}
	if !shouldStartMCPStdio() {
		t.Fatalf("expected ENABLE_MCP_STDIO=1 to enable stdio")
	}

	_ = os.Unsetenv("ENABLE_MCP_STDIO")
	os.Args = []string{"server", "--mcp-stdio"}
	if !shouldStartMCPStdio() {
		t.Fatalf("expected --mcp-stdio flag to enable stdio")
	}
}

func TestIsPipedClosedFile(t *testing.T) {
	f, err := os.CreateTemp("", "piped-test-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	defer os.Remove(name)

	if isPiped(f) {
		t.Fatalf("expected closed file to return false")
	}
}

func TestShouldStartMCPStdioNoMCPArgs(t *testing.T) {
	oldArgs := os.Args
	oldEnv, hadEnv := os.LookupEnv("ENABLE_MCP_STDIO")
	defer func() {
		os.Args = oldArgs
		if hadEnv {
			_ = os.Setenv("ENABLE_MCP_STDIO", oldEnv)
		} else {
			_ = os.Unsetenv("ENABLE_MCP_STDIO")
		}
	}()

	_ = os.Unsetenv("ENABLE_MCP_STDIO")
	os.Args = []string{"server", "--other-flag"}
	// Exercises the arg loop fallback and isPiped code path.
	_ = shouldStartMCPStdio()
}
