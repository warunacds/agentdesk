package skills

import (
	"strings"
)

// ValidationWarning represents a single validation issue found in a skill file.
type ValidationWarning struct {
	Path    string `json:"path"`
	Type    string `json:"type"`    // "missing-name", "malformed-frontmatter", "large-file", "empty-file", "missing-description"
	Message string `json:"message"`
}

const largeFileThreshold = 100 * 1024 // 100KB

// ValidateSkill checks a single skill for common issues and returns a list
// of warnings. The checks include: empty file, large file, malformed frontmatter,
// missing name, and missing description.
func ValidateSkill(skill Skill) []ValidationWarning {
	var warnings []ValidationWarning

	// Empty file
	if skill.Size == 0 || strings.TrimSpace(skill.Content) == "" {
		warnings = append(warnings, ValidationWarning{
			Path:    skill.Path,
			Type:    "empty-file",
			Message: "File is empty",
		})
		return warnings // no point checking further
	}

	// Large file
	if skill.Size > largeFileThreshold {
		warnings = append(warnings, ValidationWarning{
			Path:    skill.Path,
			Type:    "large-file",
			Message: "File exceeds 100KB — may be slow to sync or edit",
		})
	}

	// Malformed frontmatter: starts with "---" but never closes
	if strings.HasPrefix(skill.Content, "---") {
		rest := skill.Content[3:]
		if !strings.Contains(rest, "\n---") {
			warnings = append(warnings, ValidationWarning{
				Path:    skill.Path,
				Type:    "malformed-frontmatter",
				Message: "Frontmatter block is not closed (missing closing ---)",
			})
		}
	}

	// Missing name in frontmatter
	if skill.Frontmatter["name"] == "" {
		warnings = append(warnings, ValidationWarning{
			Path:    skill.Path,
			Type:    "missing-name",
			Message: "No 'name' field in frontmatter",
		})
	}

	// Missing description (info-level, not a hard warning)
	if skill.Frontmatter["description"] == "" {
		warnings = append(warnings, ValidationWarning{
			Path:    skill.Path,
			Type:    "missing-description",
			Message: "No 'description' field in frontmatter",
		})
	}

	return warnings
}

// ValidateAll runs validation on every skill and returns warnings grouped
// by file path. Only files with at least one warning are included in the map.
func ValidateAll(allSkills []Skill) map[string][]ValidationWarning {
	result := make(map[string][]ValidationWarning)

	for _, skill := range allSkills {
		warnings := ValidateSkill(skill)
		if len(warnings) > 0 {
			result[skill.Path] = warnings
		}
	}

	return result
}
