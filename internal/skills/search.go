package skills

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxSearchResults = 100

// SearchResult represents a single line match from a full-text search.
type SearchResult struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Tool    string `json:"tool"`
	Line    int    `json:"line"`
	Content string `json:"content"` // the matching line with context
	Match   string `json:"match"`   // the matched substring
}

// SearchFiles performs a case-insensitive full-text search across all scanned
// skill files. It returns matching lines with line numbers, capped at 100 results.
// Binary files are skipped.
func SearchFiles(query string, extraDirs []string) []SearchResult {
	if query == "" {
		return []SearchResult{}
	}

	allSkills := Scan(extraDirs)
	lowerQuery := strings.ToLower(query)

	var results []SearchResult

	for _, sk := range allSkills {
		if len(results) >= maxSearchResults {
			break
		}

		matches := searchInFile(sk.Path, sk.Name, string(sk.Tool), lowerQuery)
		for _, m := range matches {
			if len(results) >= maxSearchResults {
				break
			}
			results = append(results, m)
		}
	}

	return results
}

// searchInFile reads a file line-by-line and returns matches for the given
// lowercased query. Skips files that appear to be binary.
func searchInFile(path, name, tool, lowerQuery string) []SearchResult {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Quick binary check: read a small sample and check for valid UTF-8
	// with no null bytes.
	if isBinaryFile(path) {
		return nil
	}

	var results []SearchResult
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 256*1024), 256*1024) // 256KB line buffer
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lowerLine := strings.ToLower(line)

		idx := strings.Index(lowerLine, lowerQuery)
		if idx < 0 {
			continue
		}

		// Extract the actual matched substring (preserving original case)
		matchEnd := idx + len(lowerQuery)
		if matchEnd > len(line) {
			matchEnd = len(line)
		}
		matched := line[idx:matchEnd]

		// Trim very long lines for display
		content := line
		if len(content) > 200 {
			// Show context around the match
			start := idx - 40
			if start < 0 {
				start = 0
			}
			end := matchEnd + 40
			if end > len(content) {
				end = len(content)
			}
			content = content[start:end]
		}

		results = append(results, SearchResult{
			Path:    path,
			Name:    name,
			Tool:    tool,
			Line:    lineNum,
			Content: strings.TrimSpace(content),
			Match:   matched,
		})
	}

	return results
}

// isBinaryFile checks whether a file appears to be binary by reading the first
// 512 bytes and looking for null bytes or invalid UTF-8 sequences.
func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return true
	}
	buf = buf[:n]

	// Check for null bytes
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}

	// Check for valid UTF-8
	if !utf8.Valid(buf) {
		return true
	}

	// Check file extension — skip known binary extensions
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".ico": true, ".bmp": true, ".webp": true, ".svg": true,
		".zip": true, ".tar": true, ".gz": true, ".bz2": true,
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".pdf": true, ".doc": true, ".docx": true,
		".woff": true, ".woff2": true, ".ttf": true, ".otf": true,
		".mp3": true, ".mp4": true, ".wav": true, ".avi": true,
	}

	return binaryExts[ext]
}
