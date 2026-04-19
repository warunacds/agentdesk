package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// searchInFile (unexported helper, tested via package-level access)
// ---------------------------------------------------------------------------

func TestSearchInFile(t *testing.T) {
	dir := t.TempDir()

	// Create a text file with known content.
	path := filepath.Join(dir, "test.md")
	content := "line one\nfind this term here\nline three\nAnother find this match"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	results := searchInFile(path, "test", "claude-code", "find this")
	if len(results) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(results))
	}

	// First match should be on line 2.
	if results[0].Line != 2 {
		t.Errorf("first result Line = %d, want 2", results[0].Line)
	}
	if !strings.Contains(results[0].Content, "find this term") {
		t.Errorf("first result Content = %q, expected it to contain 'find this term'", results[0].Content)
	}
	if results[0].Match != "find this" {
		t.Errorf("first result Match = %q, want %q", results[0].Match, "find this")
	}
	if results[0].Name != "test" {
		t.Errorf("first result Name = %q, want %q", results[0].Name, "test")
	}
	if results[0].Tool != "claude-code" {
		t.Errorf("first result Tool = %q, want %q", results[0].Tool, "claude-code")
	}

	// Second match should be on line 4.
	if results[1].Line != 4 {
		t.Errorf("second result Line = %d, want 4", results[1].Line)
	}
}

func TestSearchInFile_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "case.md")
	content := "HELLO World\nhello world\nHeLLo WoRLD"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// searchInFile expects the query to already be lowercased.
	results := searchInFile(path, "case", "test", "hello world")
	if len(results) != 3 {
		t.Fatalf("expected 3 case-insensitive matches, got %d", len(results))
	}
}

func TestSearchInFile_NoMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty_match.md")
	content := "nothing relevant here\njust some text"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	results := searchInFile(path, "empty_match", "test", "nonexistent term")
	if len(results) != 0 {
		t.Errorf("expected 0 matches, got %d", len(results))
	}
}

func TestSearchInFile_NonexistentFile(t *testing.T) {
	results := searchInFile("/nonexistent/path/file.md", "nofile", "test", "query")
	if results != nil {
		t.Errorf("expected nil for nonexistent file, got %v", results)
	}
}

func TestSearchInFile_LongLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.md")

	// Build a line longer than 200 chars with the match in the middle.
	prefix := strings.Repeat("x", 120)
	suffix := strings.Repeat("y", 120)
	longLine := prefix + "FINDME" + suffix
	if err := os.WriteFile(path, []byte(longLine), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	results := searchInFile(path, "long", "test", "findme")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}

	// Content should be trimmed — shorter than the full line.
	if len(results[0].Content) >= len(longLine) {
		t.Errorf("expected truncated content for long line, got length %d", len(results[0].Content))
	}
}

// ---------------------------------------------------------------------------
// isBinaryFile
// ---------------------------------------------------------------------------

func TestIsBinaryFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("text file", func(t *testing.T) {
		path := filepath.Join(dir, "text.md")
		if err := os.WriteFile(path, []byte("plain text content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if isBinaryFile(path) {
			t.Error("expected text file to not be detected as binary")
		}
	})

	t.Run("binary file with null bytes", func(t *testing.T) {
		path := filepath.Join(dir, "binary.bin")
		data := []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F, 0x00, 0x57, 0x6F, 0x72, 0x6C, 0x64}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if !isBinaryFile(path) {
			t.Error("expected file with null bytes to be detected as binary")
		}
	})

	t.Run("binary extension", func(t *testing.T) {
		path := filepath.Join(dir, "image.png")
		if err := os.WriteFile(path, []byte("not really a png but has the extension"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		if !isBinaryFile(path) {
			t.Error("expected .png file to be detected as binary by extension")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		if !isBinaryFile(filepath.Join(dir, "does-not-exist.txt")) {
			t.Error("expected nonexistent file to be treated as binary")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(dir, "empty.txt")
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		// Empty file: Read returns 0 bytes, treated as binary.
		if !isBinaryFile(path) {
			t.Error("expected empty file to be treated as binary")
		}
	})
}

// ---------------------------------------------------------------------------
// SearchFiles (integration-level, uses Scan internally)
// ---------------------------------------------------------------------------

func TestSearchFiles_EmptyQuery(t *testing.T) {
	results := SearchFiles("", nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}
