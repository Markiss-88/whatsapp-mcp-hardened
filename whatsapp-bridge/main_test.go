package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMediaPath(t *testing.T) {
	baseDir := t.TempDir()
	root := filepath.Join(baseDir, "outbox")
	t.Setenv("WHATSAPP_MCP_MEDIA_DIR", root)

	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}

	insideFile := filepath.Join(root, "inside.txt")
	if err := os.WriteFile(insideFile, []byte("inside"), 0644); err != nil {
		t.Fatal(err)
	}

	// validateMediaPath returns a symlink-resolved path, and on macOS the temp
	// dir lives under /var -> /private/var, so compare against the resolved form.
	resolvedInsideFile, err := filepath.EvalSymlinks(insideFile)
	if err != nil {
		t.Fatal(err)
	}

	outsideFile := filepath.Join(baseDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(root, "outside-link")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	siblingDir := root + "-evil"
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		t.Fatal(err)
	}
	siblingFile := filepath.Join(siblingDir, "sibling.txt")
	if err := os.WriteFile(siblingFile, []byte("sibling"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "inside root", path: insideFile},
		{name: "traversal", path: filepath.Join(root, "..", filepath.Base(outsideFile)), wantErr: true},
		{name: "absolute outside root", path: outsideFile, wantErr: true},
		{name: "symlink escape", path: symlinkPath, wantErr: true},
		{name: "sibling prefix", path: siblingFile, wantErr: true},
		{name: "directory", path: root, wantErr: true},
		{name: "empty", path: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := validateMediaPath(test.path)
			if test.wantErr {
				if err == nil {
					t.Fatalf("validateMediaPath(%q) succeeded with %q", test.path, got)
				}
				if !strings.Contains(err.Error(), root) {
					t.Fatalf("error %q does not name allowed root %q", err, root)
				}
				return
			}

			if err != nil {
				t.Fatalf("validateMediaPath(%q) returned error: %v", test.path, err)
			}
			if got != resolvedInsideFile {
				t.Fatalf("validateMediaPath(%q) = %q, want %q", test.path, got, resolvedInsideFile)
			}
		})
	}
}

func TestSanitizeDownloadFilename(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		want         string
		wantFallback bool
	}{
		{name: "traversal", input: "../../../etc/passwd", want: "passwd"},
		{name: "plain", input: "report.pdf", want: "report.pdf"},
		{name: "empty", input: "", wantFallback: true},
		{name: "dot", input: ".", wantFallback: true},
		{name: "dot dot", input: "..", wantFallback: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sanitizeDownloadFilename(test.input, "document", "message-id")
			if test.wantFallback {
				if got == "" {
					t.Fatal("fallback filename is empty")
				}
				if strings.ContainsRune(got, os.PathSeparator) {
					t.Fatalf("fallback filename %q contains path separator", got)
				}
				return
			}
			if got != test.want {
				t.Fatalf("sanitizeDownloadFilename(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
