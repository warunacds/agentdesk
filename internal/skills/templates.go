package skills

// SkillTemplate represents a built-in template that users can create new skills from.
type SkillTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tool        ToolType `json:"tool"`
	Category    string   `json:"category"` // "agent", "rule", etc.
	Content     string   `json:"content"`
}

// BuiltinTemplates is the list of pre-defined templates available in Forge.
var BuiltinTemplates = []SkillTemplate{
	{
		ID:          "claude-agent-react",
		Name:        "Claude Code Agent (React)",
		Description: "A Claude Code agent specialised for React/TypeScript projects",
		Tool:        ToolClaudeCode,
		Category:    "agent",
		Content: `---
name: React Agent
description: Claude Code agent for React/TypeScript development
---

# React Agent

You are a senior React/TypeScript engineer. Follow these guidelines:

## Stack
- React 18+ with functional components and hooks
- TypeScript in strict mode
- CSS Modules or Tailwind CSS for styling

## Conventions
- Use named exports for components
- Keep components small and focused (< 200 lines)
- Use custom hooks to extract reusable logic
- Prefer composition over prop drilling — use Context where appropriate

## Code Style
- Use ` + "`" + `const` + "`" + ` for component definitions: ` + "`" + `const MyComponent: React.FC<Props> = ...` + "`" + `
- Destructure props in the function signature
- Use early returns for guard clauses
- Write meaningful variable and function names

## Testing
- Write tests with React Testing Library
- Test behavior, not implementation details
- Use ` + "`" + `screen.getByRole` + "`" + ` and ` + "`" + `screen.getByText` + "`" + ` over ` + "`" + `getByTestId` + "`" + `

## Error Handling
- Use Error Boundaries for component-level errors
- Handle async errors with try/catch in effects and handlers
- Display user-friendly error messages
`,
	},
	{
		ID:          "claude-agent-python",
		Name:        "Claude Code Agent (Python)",
		Description: "A Claude Code agent for Python backend development",
		Tool:        ToolClaudeCode,
		Category:    "agent",
		Content: `---
name: Python Agent
description: Claude Code agent for Python development
---

# Python Agent

You are a senior Python engineer. Follow these guidelines:

## Stack
- Python 3.11+
- Type hints on all function signatures
- Virtual environments with venv or poetry

## Code Style
- Follow PEP 8 and PEP 257 (docstrings)
- Use f-strings for string formatting
- Prefer pathlib over os.path
- Use dataclasses or Pydantic for structured data

## Architecture
- Keep functions pure where possible
- Use dependency injection for testability
- Separate business logic from I/O
- Use async/await for I/O-bound operations

## Testing
- Use pytest with fixtures
- Write parametrized tests for multiple cases
- Aim for meaningful test coverage
- Mock external services at the boundary

## Error Handling
- Use specific exception types
- Never use bare except clauses
- Log errors with structured context
- Return meaningful error messages to callers
`,
	},
	{
		ID:          "claude-agent-go",
		Name:        "Claude Code Agent (Go)",
		Description: "A Claude Code agent for Go backend development",
		Tool:        ToolClaudeCode,
		Category:    "agent",
		Content: `---
name: Go Agent
description: Claude Code agent for Go development
---

# Go Agent

You are a senior Go engineer. Follow these guidelines:

## Stack
- Go 1.22+
- Standard library preferred over third-party packages
- Use modules for dependency management

## Code Style
- Follow Effective Go and the Go Code Review Comments wiki
- Use gofmt and go vet
- Keep functions focused — one function, one responsibility
- Use mixedCaps naming, not underscores

## Architecture
- Accept interfaces, return concrete types
- Keep interfaces small (1-3 methods)
- Use composition over inheritance
- Define interfaces in the consuming package

## Error Handling
- Always handle errors — never discard with _
- Wrap errors with context: fmt.Errorf("context: %w", err)
- Use custom error types for distinguishable errors
- Return errors to the caller — let them decide how to handle

## Testing
- Table-driven tests for multiple cases
- Use testing.T.Run for subtests
- Write integration tests with build tags
- Use interfaces and mocks to isolate units

## Concurrency
- Use goroutines and channels idiomatically
- Always pass context.Context for cancellation
- Use sync.Mutex for shared state, not channels
- Watch for goroutine leaks — ensure they can exit
`,
	},
	{
		ID:          "claude-agent-general",
		Name:        "Claude Code Agent (General)",
		Description: "A general-purpose Claude Code agent",
		Tool:        ToolClaudeCode,
		Category:    "agent",
		Content: `---
name: General Agent
description: General-purpose Claude Code agent
---

# General Agent

You are a senior software engineer. Follow these guidelines:

## Approach
- Understand the problem before writing code
- Ask clarifying questions when requirements are ambiguous
- Prefer simple, readable solutions over clever ones
- Write code that is easy to maintain and extend

## Code Quality
- Write clear, self-documenting code
- Keep functions small and focused
- Use meaningful names for variables and functions
- Add comments only when the "why" is not obvious

## Testing
- Write tests for all business logic
- Test edge cases and error paths
- Keep tests readable and maintainable

## Error Handling
- Handle all error cases explicitly
- Provide helpful error messages
- Log errors with sufficient context for debugging

## Documentation
- Document public APIs
- Keep documentation up to date with code
- Include examples where helpful
`,
	},
	{
		ID:          "cursor-rule-nextjs",
		Name:        "Cursor Rule (Next.js)",
		Description: "Cursor rules for Next.js App Router projects",
		Tool:        ToolCursor,
		Category:    "rule",
		Content: `# Next.js Rules

You are an expert in Next.js 14+ with the App Router.

## Framework Conventions
- Use the App Router (app/ directory) — not Pages Router
- Use Server Components by default; add "use client" only when needed
- Use Server Actions for mutations
- Use route handlers (route.ts) for API endpoints

## Data Fetching
- Fetch data in Server Components using async/await
- Use React Suspense for loading states
- Implement error boundaries with error.tsx files
- Use generateStaticParams for static generation

## Styling
- Use Tailwind CSS with the cn() utility for conditional classes
- Follow mobile-first responsive design
- Use CSS variables for theming

## Performance
- Use next/image for optimized images
- Implement proper metadata with generateMetadata
- Use dynamic imports for code splitting
- Minimize client-side JavaScript

## File Structure
- Colocate related files within route segments
- Use (groups) for layout organization
- Keep components in a components/ directory
- Use lib/ for utilities and shared logic
`,
	},
	{
		ID:          "cursor-rule-typescript",
		Name:        "Cursor Rule (TypeScript)",
		Description: "Cursor rules for TypeScript projects",
		Tool:        ToolCursor,
		Category:    "rule",
		Content: `# TypeScript Rules

You are an expert in TypeScript development.

## Type Safety
- Enable strict mode in tsconfig.json
- Avoid using 'any' — use 'unknown' with type guards instead
- Use discriminated unions for complex state
- Prefer interfaces for object shapes, types for unions and utilities

## Code Style
- Use const assertions for literal types
- Prefer readonly for immutable data
- Use template literal types where appropriate
- Leverage utility types (Pick, Omit, Partial, Required)

## Error Handling
- Use Result types or discriminated unions for error handling
- Type your error handlers — avoid catch(e: any)
- Define custom error classes with proper typing

## Patterns
- Use generics to create reusable, type-safe utilities
- Prefer function overloads for complex signatures
- Use branded types for semantic type safety
- Implement exhaustive checks with never

## Testing
- Use vitest or jest with @types
- Type your test fixtures and mocks
- Test type-level logic with type assertion tests
`,
	},
	{
		ID:          "cursor-rule-python",
		Name:        "Cursor Rule (Python)",
		Description: "Cursor rules for Python projects",
		Tool:        ToolCursor,
		Category:    "rule",
		Content: `# Python Rules

You are an expert in Python development.

## Style
- Follow PEP 8 style guide
- Use type hints on all function signatures and variables
- Use docstrings (Google style) on all public functions and classes
- Prefer f-strings over format() or %

## Architecture
- Use dataclasses or Pydantic models for structured data
- Prefer composition over inheritance
- Keep functions pure where possible
- Use dependency injection for external services

## Best Practices
- Use pathlib for file operations
- Use contextmanagers for resource management
- Prefer list/dict/set comprehensions over loops when clear
- Use enumerate() instead of manual index tracking

## Error Handling
- Use specific exception types — never bare except
- Create custom exceptions for domain errors
- Use logging module instead of print for production code

## Testing
- Use pytest with fixtures
- Write parametrized tests for multiple inputs
- Mock at the boundary, not internally
- Aim for high coverage on business logic
`,
	},
	{
		ID:          "gemini-agent",
		Name:        "Gemini CLI Agent",
		Description: "A Gemini CLI agent configuration",
		Tool:        ToolGeminiCLI,
		Category:    "agent",
		Content: `# Gemini CLI Agent

## General Instructions

- Follow the project's existing code style and conventions
- Write clean, maintainable code with clear intent
- Ask clarifying questions before making assumptions
- Prefer simple solutions over complex ones

## Coding Style

- Use consistent naming conventions
- Keep functions focused and small
- Write meaningful variable names
- Add comments for non-obvious logic

## Error Handling

- Handle all error cases explicitly
- Provide helpful error messages
- Log errors with context for debugging

## Testing

- Write tests for new functionality
- Cover edge cases and error paths
- Keep tests independent and repeatable
`,
	},
	{
		ID:          "kiro-steering",
		Name:        "Kiro Steering File",
		Description: "A Kiro steering file for project guidance",
		Tool:        ToolKiro,
		Category:    "rule",
		Content: `---
name: Project Steering
description: Kiro steering file for project conventions
inclusion: auto
---

# Project Steering

## Project Overview
Describe your project here.

## Tech Stack
- List your technologies

## Coding Conventions
- Follow existing patterns in the codebase
- Use consistent naming conventions
- Keep functions small and focused

## File Structure
- Describe your project's directory layout

## Testing
- Write tests for all new functionality
- Follow existing test patterns

## Deployment
- Describe your deployment process
`,
	},
	{
		ID:          "codex-agent",
		Name:        "Codex Agent",
		Description: "An OpenAI Codex agent configuration",
		Tool:        ToolCodex,
		Category:    "agent",
		Content: `# Codex Agent

## Instructions

You are a helpful coding assistant. Follow these guidelines:

## Approach
- Read and understand the existing codebase before making changes
- Follow the project's established patterns and conventions
- Write clean, maintainable code
- Test your changes

## Code Quality
- Write self-documenting code with clear names
- Keep functions focused on a single task
- Handle errors gracefully
- Add comments only when the intent is not obvious

## Communication
- Explain your reasoning when making decisions
- Ask for clarification when requirements are unclear
- Flag potential issues or trade-offs
`,
	},
}

// FindTemplate looks up a template by its ID. Returns nil if not found.
func FindTemplate(id string) *SkillTemplate {
	for i := range BuiltinTemplates {
		if BuiltinTemplates[i].ID == id {
			return &BuiltinTemplates[i]
		}
	}
	return nil
}
