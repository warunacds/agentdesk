package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseFrontmatter
// ---------------------------------------------------------------------------

func TestParseFrontmatter_WithFrontmatter(t *testing.T) {
	content := "---\nname: My Skill\ndescription: Does things\n---\n\n# Body"
	fm, body := ParseFrontmatter(content)

	if fm["name"] != "My Skill" {
		t.Errorf("expected name='My Skill', got %q", fm["name"])
	}
	if fm["description"] != "Does things" {
		t.Errorf("expected description='Does things', got %q", fm["description"])
	}
	if !strings.Contains(body, "# Body") {
		t.Errorf("expected body to contain '# Body', got %q", body)
	}
}

func TestParseFrontmatter_WithoutFrontmatter(t *testing.T) {
	content := "# Just markdown\n\nSome text here."
	fm, body := ParseFrontmatter(content)

	if len(fm) != 0 {
		t.Errorf("expected empty frontmatter, got %v", fm)
	}
	if body != content {
		t.Errorf("expected body to equal original content")
	}
}

func TestParseFrontmatter_Empty(t *testing.T) {
	fm, body := ParseFrontmatter("")

	if len(fm) != 0 {
		t.Errorf("expected empty frontmatter for empty string, got %v", fm)
	}
	if body != "" {
		t.Errorf("expected empty body for empty string, got %q", body)
	}
}

func TestParseFrontmatter_Malformed_UnclosedDelimiter(t *testing.T) {
	content := "---\nname: Broken\nNo closing delimiter"
	fm, body := ParseFrontmatter(content)

	// When the closing --- is missing, the parser returns the original content
	// with an empty frontmatter map.
	if len(fm) != 0 {
		t.Errorf("expected empty frontmatter for malformed input, got %v", fm)
	}
	if body != content {
		t.Errorf("expected body to equal original content for malformed input")
	}
}

func TestParseFrontmatter_QuotedValues(t *testing.T) {
	content := "---\nname: \"My Quoted Skill\"\ndescription: 'single quoted'\n---\nBody"
	fm, _ := ParseFrontmatter(content)

	if fm["name"] != "My Quoted Skill" {
		t.Errorf("expected stripped double-quoted name, got %q", fm["name"])
	}
	if fm["description"] != "single quoted" {
		t.Errorf("expected stripped single-quoted description, got %q", fm["description"])
	}
}

func TestParseFrontmatter_ValueWithColon(t *testing.T) {
	// Only the first colon is used as the key-value separator.
	content := "---\nname: key: with colon\n---\nBody"
	fm, _ := ParseFrontmatter(content)

	if fm["name"] != "key: with colon" {
		t.Errorf("expected value with colon preserved, got %q", fm["name"])
	}
}

// ---------------------------------------------------------------------------
// ReplaceFrontmatter
// ---------------------------------------------------------------------------

func TestReplaceFrontmatter_AddNew(t *testing.T) {
	content := "# Hello World"
	result := ReplaceFrontmatter(content, map[string]string{"name": "Test"})

	if !strings.HasPrefix(result, "---\n") {
		t.Errorf("expected result to start with frontmatter delimiter")
	}
	if !strings.Contains(result, "name: Test") {
		t.Errorf("expected result to contain 'name: Test'")
	}
	if !strings.Contains(result, "# Hello World") {
		t.Errorf("expected result to preserve body content")
	}
}

func TestReplaceFrontmatter_UpdateExisting(t *testing.T) {
	content := "---\nname: Old\ndescription: Old desc\n---\n# Body"
	result := ReplaceFrontmatter(content, map[string]string{
		"name":        "New",
		"description": "New desc",
	})

	if !strings.Contains(result, "name: New") {
		t.Errorf("expected updated name")
	}
	if !strings.Contains(result, "description: New desc") {
		t.Errorf("expected updated description")
	}
	if !strings.Contains(result, "# Body") {
		t.Errorf("expected body preserved")
	}
}

func TestReplaceFrontmatter_PreservesOrder(t *testing.T) {
	content := "---\ndescription: Desc\nname: Name\n---\nBody"
	result := ReplaceFrontmatter(content, map[string]string{
		"name":        "Name",
		"description": "Desc",
	})

	// Existing order should be preserved: description before name.
	descIdx := strings.Index(result, "description:")
	nameIdx := strings.Index(result, "name:")

	if descIdx > nameIdx {
		t.Errorf("expected description before name (preserving original order), descIdx=%d nameIdx=%d", descIdx, nameIdx)
	}
}

func TestReplaceFrontmatter_AddsNewKeysAtEnd(t *testing.T) {
	content := "---\nname: Name\n---\nBody"
	result := ReplaceFrontmatter(content, map[string]string{
		"name":    "Name",
		"version": "1.0",
	})

	nameIdx := strings.Index(result, "name:")
	versionIdx := strings.Index(result, "version:")

	if versionIdx < nameIdx {
		t.Errorf("expected new key 'version' after existing key 'name'")
	}
}

func TestReplaceFrontmatter_EmptyMap(t *testing.T) {
	content := "# No changes expected"
	result := ReplaceFrontmatter(content, map[string]string{})

	if result != content {
		t.Errorf("expected no changes for empty map, got %q", result)
	}
}

func TestReplaceFrontmatter_NoExistingFrontmatter(t *testing.T) {
	content := "Just plain text without frontmatter"
	result := ReplaceFrontmatter(content, map[string]string{"name": "Added"})

	if !strings.HasPrefix(result, "---\n") {
		t.Errorf("expected frontmatter block to be prepended")
	}
	if !strings.Contains(result, "name: Added") {
		t.Errorf("expected 'name: Added' in result")
	}
	if !strings.Contains(result, "Just plain text without frontmatter") {
		t.Errorf("expected original content preserved as body")
	}
}

func TestReplaceFrontmatter_SpecialCharsQuoted(t *testing.T) {
	content := "# Body"
	result := ReplaceFrontmatter(content, map[string]string{
		"url": "https://example.com:8080/path",
	})

	// Values with colons should be quoted.
	if !strings.Contains(result, `url: "https://example.com:8080/path"`) {
		t.Errorf("expected URL to be quoted due to colon, got:\n%s", result)
	}
}

// ---------------------------------------------------------------------------
// GenerateTemplate
// ---------------------------------------------------------------------------

func TestGenerateTemplate(t *testing.T) {
	tests := []struct {
		tool     ToolType
		contains string
	}{
		{ToolClaudeCode, "---\nname:"},
		{ToolGeminiCLI, "## General Instructions"},
		{ToolKiro, "inclusion: auto"},
		{ToolCursor, "## Guidelines"},
		{ToolWindsurf, "## Guidelines"},
		{ToolCopilot, "# Copilot Instructions"},
		{ToolCodex, "# Agents"},
		{ToolUnknown, "# TestSkill"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tool), func(t *testing.T) {
			result := GenerateTemplate(tt.tool, "TestSkill")
			if !strings.Contains(result, tt.contains) {
				t.Errorf("GenerateTemplate(%q) missing expected content %q, got:\n%s", tt.tool, tt.contains, result)
			}
		})
	}
}

func TestGenerateTemplate_IncludesName(t *testing.T) {
	// Every template should include the name somewhere.
	tools := []ToolType{
		ToolClaudeCode, ToolGeminiCLI, ToolKiro, ToolCursor,
		ToolWindsurf, ToolCopilot, ToolCodex, ToolUnknown,
	}
	for _, tool := range tools {
		t.Run(string(tool), func(t *testing.T) {
			result := GenerateTemplate(tool, "MySpecialName")
			if !strings.Contains(result, "MySpecialName") {
				t.Errorf("GenerateTemplate(%q) does not contain the name 'MySpecialName'", tool)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DefaultFilename
// ---------------------------------------------------------------------------

func TestDefaultFilename(t *testing.T) {
	tests := []struct {
		tool     ToolType
		expected string
	}{
		{ToolClaudeCode, "CLAUDE.md"},
		{ToolGeminiCLI, "GEMINI.md"},
		{ToolKiro, "steering.md"},
		{ToolCursor, ".cursorrules"},
		{ToolWindsurf, ".windsurfrules"},
		{ToolCopilot, "copilot-instructions.md"},
		{ToolAider, "aider.conventions.md"},
		{ToolAmp, "skill.md"},
		{ToolCodex, "AGENTS.md"},
		{ToolUnknown, "skill.md"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tool), func(t *testing.T) {
			got := DefaultFilename(tt.tool)
			if got != tt.expected {
				t.Errorf("DefaultFilename(%q) = %q, want %q", tt.tool, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DefaultSubdir
// ---------------------------------------------------------------------------

func TestDefaultSubdir(t *testing.T) {
	tests := []struct {
		tool     ToolType
		expected string
	}{
		{ToolKiro, ".kiro/steering"},
		{ToolCopilot, ".github"},
		{ToolAmp, ".amp/skills"},
		{ToolCursor, ".cursor/rules"},
		{ToolClaudeCode, ""},
		{ToolGeminiCLI, ""},
		{ToolWindsurf, ""},
		{ToolAider, ""},
		{ToolCodex, ""},
		{ToolUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.tool), func(t *testing.T) {
			got := DefaultSubdir(tt.tool)
			if got != tt.expected {
				t.Errorf("DefaultSubdir(%q) = %q, want %q", tt.tool, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Folder-based skills: Skill struct fields
// ---------------------------------------------------------------------------

func TestSkill_HasDirectoryAndAuxiliaryFilesFields(t *testing.T) {
	s := Skill{
		Directory: "/tmp/foo",
		AuxiliaryFiles: []AuxFile{
			{Path: "/tmp/foo/a.sh", RelPath: "a.sh", Type: "script"},
		},
	}
	if s.Directory != "/tmp/foo" {
		t.Fatal("Directory field not set")
	}
	if len(s.AuxiliaryFiles) != 1 {
		t.Fatal("AuxiliaryFiles field not set")
	}
	if s.AuxiliaryFiles[0].Type != "script" {
		t.Fatalf("Type not populated: %+v", s.AuxiliaryFiles[0])
	}
}

// ---------------------------------------------------------------------------
// Folder-based skills: Scanner detection
// ---------------------------------------------------------------------------

func TestScanner_DetectsFolderSkillWithAuxFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	skillDir := filepath.Join(tmp, ".claude", "skills", "my-skill")
	scriptsDir := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Foo\n---\n# Foo"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "setup.sh"), []byte("#!/bin/bash"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "examples.md"), []byte("# Examples"), 0644); err != nil {
		t.Fatal(err)
	}

	result := Scan(nil)

	var found *Skill
	for i := range result {
		if filepath.Base(result[i].Path) == "SKILL.md" {
			found = &result[i]
			break
		}
	}
	if found == nil {
		t.Fatal("SKILL.md not found in scan results")
	}
	if found.Directory != skillDir {
		t.Fatalf("Directory mismatch: got %q want %q", found.Directory, skillDir)
	}
	if len(found.AuxiliaryFiles) != 2 {
		t.Fatalf("expected 2 aux files, got %d: %+v", len(found.AuxiliaryFiles), found.AuxiliaryFiles)
	}

	relPaths := map[string]string{}
	for _, a := range found.AuxiliaryFiles {
		relPaths[a.RelPath] = a.Type
	}
	if relPaths["examples.md"] != "markdown" {
		t.Fatalf("examples.md not classified as markdown: %v", relPaths)
	}
	if relPaths["scripts/setup.sh"] != "script" {
		t.Fatalf("scripts/setup.sh not classified as script: %v", relPaths)
	}
}

func TestScanner_FileSkillHasEmptyDirectory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	agentsDir := filepath.Join(tmp, ".claude", "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "foo.md"), []byte("# Foo"), 0644)

	result := Scan(nil)

	var found *Skill
	for i := range result {
		if filepath.Base(result[i].Path) == "foo.md" {
			found = &result[i]
			break
		}
	}
	if found == nil {
		t.Fatal("foo.md not found")
	}
	if found.Directory != "" {
		t.Fatalf("file skill should have empty Directory, got %q", found.Directory)
	}
	if len(found.AuxiliaryFiles) != 0 {
		t.Fatalf("file skill should have no aux files, got %d", len(found.AuxiliaryFiles))
	}
}

func TestScanner_FolderSkillExcludesDotFolders(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	skillDir := filepath.Join(tmp, ".claude", "skills", "my-skill")
	gitDir := filepath.Join(skillDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Foo"), 0644)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("x"), 0644)

	result := Scan(nil)
	var found *Skill
	for i := range result {
		if filepath.Base(result[i].Path) == "SKILL.md" {
			found = &result[i]
			break
		}
	}
	if found == nil {
		t.Fatal("SKILL.md not found")
	}
	for _, a := range found.AuxiliaryFiles {
		if strings.Contains(a.RelPath, ".git") {
			t.Fatalf(".git files should be excluded: %+v", a)
		}
	}
}
