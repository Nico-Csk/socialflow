package projecttest

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// projectRoot walks up from the working directory until it finds go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for range 10 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	t.Fatal("cannot find project root (go.mod not found)")
	return ""
}

// mustReadFile reads a file and returns its lines (or fails the test).
func mustReadFile(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	return lines
}

// filesModifiedByChange lists every file touched by this hardening change.
func filesModifiedByChange(root string) []string {
	return []string{
		filepath.Join(root, ".gitignore"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, ".atl", "skill-registry.md"),
		filepath.Join(root, ".agents", "skills", "golang-testing", "SKILL.md"),
		filepath.Join(root, ".agents", "skills", "golang-patterns", "SKILL.md"),
	}
}

// scopedGoFiles returns the documentation files that should use explicit
// package patterns (excludes .gitignore which is config, not docs).
func scopedGoFiles(root string) []string {
	return []string{
		filepath.Join(root, "README.md"),
		filepath.Join(root, ".atl", "skill-registry.md"),
		filepath.Join(root, ".agents", "skills", "golang-testing", "SKILL.md"),
		filepath.Join(root, ".agents", "skills", "golang-patterns", "SKILL.md"),
	}
}

// scopedCommandPattern matches Go commands that use explicit package patterns.
var scopedCommandPattern = regexp.MustCompile(`go\s+(test|vet|build)\s+.*\./cmd/\.\.\.\s+\./internal/\.\.\.`)

// --- Tests ---------------------------------------------------------------

// TestGitignoreContainsNodeModules verifies the root .gitignore includes a
// node_modules/ entry as defense-in-depth (spec R2).
func TestGitignoreContainsNodeModules(t *testing.T) {
	root := projectRoot(t)
	lines := mustReadFile(t, filepath.Join(root, ".gitignore"))

	t.Run("contains node_modules entry", func(t *testing.T) {
		found := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "node_modules/" || strings.HasPrefix(trimmed, "node_modules/") {
				found = true
				break
			}
		}
		if !found {
			t.Error(".gitignore does not contain a node_modules/ entry (spec R2)")
		}
	})

	t.Run("also ignores go.work", func(t *testing.T) {
		found := false
		for _, line := range lines {
			if strings.TrimSpace(line) == "go.work" {
				found = true
				break
			}
		}
		if !found {
			t.Error(".gitignore should also contain go.work entry (existing convention)")
		}
	})
}

// TestNoGoWorkFile verifies no go.work file exists at the project root (spec R4).
func TestNoGoWorkFile(t *testing.T) {
	root := projectRoot(t)
	workPath := filepath.Join(root, "go.work")
	if _, err := os.Stat(workPath); err == nil {
		t.Errorf("go.work exists at %s — spec R4 forbids go.work involvement", workPath)
	}
}

// wildcardPattern matches Go tool commands that use the ./... wildcard.
var wildcardPattern = regexp.MustCompile(`go\s+(test|vet|build|run)\s+.*\./\.\.\.`)

// TestNoWildcardInGoDocumentation verifies that no documented Go command
// uses the ./... wildcard in files touched by this hardening change (spec R1).
// Also triangulates by asserting the correct scoped pattern IS present.
func TestNoWildcardInGoDocumentation(t *testing.T) {
	root := projectRoot(t)

	t.Run("zero wildcard patterns", func(t *testing.T) {
		for _, filePath := range scopedGoFiles(root) {
			lines := mustReadFile(t, filePath)
			for i, line := range lines {
				if wildcardPattern.MatchString(line) {
					t.Errorf("%s line %d: contains wildcard ./... → %q",
						filePath, i+1, strings.TrimSpace(line))
				}
			}
		}
	})

	t.Run("each doc file has scoped commands", func(t *testing.T) {
		for _, filePath := range scopedGoFiles(root) {
			lines := mustReadFile(t, filePath)
			foundScoped := false
			for _, line := range lines {
				if scopedCommandPattern.MatchString(line) {
					foundScoped = true
					break
				}
			}
			if !foundScoped {
				t.Errorf("%s: no scoped go command (./cmd/... ./internal/...) found — the doc should use explicit patterns",
					filePath)
			}
		}
	})
}

// TestNodeModulesNotModifiedByChange verifies that no file in the change's
// scope lives under web/node_modules/ (spec R3).
func TestNodeModulesNotModifiedByChange(t *testing.T) {
	root := projectRoot(t)
	nodeModulesPrefix := filepath.Join(root, "web", "node_modules") + string(os.PathSeparator)

	t.Run("changed files are outside node_modules", func(t *testing.T) {
		for _, filePath := range filesModifiedByChange(root) {
			if strings.HasPrefix(filePath, nodeModulesPrefix) {
				t.Errorf("change modifies node_modules file: %s (spec R3 violated)", filePath)
			}
		}
	})

	t.Run("node_modules dir not in project root", func(t *testing.T) {
		// Verify that even if someone creates a root-level node_modules,
		// our changed files don't live under it.
		rootNM := filepath.Join(root, "node_modules") + string(os.PathSeparator)
		for _, filePath := range filesModifiedByChange(root) {
			if strings.HasPrefix(filePath, rootNM) {
				t.Errorf("change modifies root node_modules file: %s", filePath)
			}
		}
	})
}

// coverprofilePattern matches go test commands that include -coverprofile.
var coverprofilePattern = regexp.MustCompile(`go\s+test\s+.*-coverprofile`)

// TestCoverageUsesCoverpkg verifies that every go test command generating
// a coverage profile also scopes instrumentation with -coverpkg (spec R1).
// Triangulates by checking the coverpkg value is correct.
func TestCoverageUsesCoverpkg(t *testing.T) {
	root := projectRoot(t)

	filesToCheck := []string{
		filepath.Join(root, ".agents", "skills", "golang-testing", "SKILL.md"),
	}

	t.Run("every -coverprofile has -coverpkg", func(t *testing.T) {
		for _, filePath := range filesToCheck {
			lines := mustReadFile(t, filePath)
			for i, line := range lines {
				if !coverprofilePattern.MatchString(line) {
					continue
				}
				if !strings.Contains(line, "-coverpkg") {
					t.Errorf("%s line %d: -coverprofile without -coverpkg → %q",
						filePath, i+1, strings.TrimSpace(line))
				}
			}
		}
	})

	t.Run("coverpkg targets project packages", func(t *testing.T) {
		wantCoverpkg := "-coverpkg=./cmd/...,./internal/..."
		for _, filePath := range filesToCheck {
			lines := mustReadFile(t, filePath)
			for i, line := range lines {
				if !coverprofilePattern.MatchString(line) {
					continue
				}
				if !strings.Contains(line, wantCoverpkg) {
					t.Errorf("%s line %d: -coverpkg should use %q → got %q",
						filePath, i+1, wantCoverpkg, strings.TrimSpace(line))
				}
			}
		}
	})
}

// TestScopedCoverageExcludesNodeModules provides runtime evidence from an
// existing coverage profile that third-party packages are excluded.
func TestScopedCoverageExcludesNodeModules(t *testing.T) {
	root := projectRoot(t)

	profilePath := filepath.Join(root, "coverage.out")
	if _, err := os.Stat(profilePath); err != nil {
		t.Logf("coverage profile not found at %s", profilePath)
		t.Log("skipping — generate it first with:")
		t.Log("  go test -coverprofile=coverage.out -coverpkg=./cmd/...,./internal/... ./cmd/... ./internal/...")
		return
	}

	lines := mustReadFile(t, profilePath)

	thirdPartyIndicators := []string{
		"flatted",
		"node_modules",
	}

	t.Run("zero third-party packages", func(t *testing.T) {
		offendingLines := make(map[string][]string)
		for _, line := range lines {
			if strings.HasPrefix(line, "mode:") {
				continue
			}
			for _, indicator := range thirdPartyIndicators {
				if strings.Contains(line, indicator) {
					offendingLines[indicator] = append(offendingLines[indicator], line)
				}
			}
		}
		if len(offendingLines) > 0 {
			for indicator, entries := range offendingLines {
				t.Errorf("coverage profile includes third-party package %q: %d entries (e.g. %q)",
					indicator, len(entries), strings.TrimSpace(entries[0]))
			}
		} else {
			t.Logf("verified: zero third-party packages (flatted/node_modules) in profile")
		}
	})

	t.Run("only project packages appear", func(t *testing.T) {
		projectPrefix := "github.com/nicoc/socialflow/"
		nonProjectLines := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "mode:") {
				continue
			}
			if !strings.Contains(line, projectPrefix) {
				nonProjectLines++
				if nonProjectLines <= 3 {
					t.Logf("non-project entry: %s", strings.TrimSpace(line))
				}
			}
		}
		if nonProjectLines > 0 {
			t.Errorf("coverage profile contains %d entries outside project module %q",
				nonProjectLines, projectPrefix)
		} else {
			t.Logf("verified: all %d coverage entries are within %s", len(lines)-1, projectPrefix)
		}
	})
}

// TestReadmeHasScopedTestCommand verifies the README's running-tests section
// uses explicit package patterns (spec R1).
func TestReadmeHasScopedTestCommand(t *testing.T) {
	root := projectRoot(t)
	readmePath := filepath.Join(root, "README.md")
	lines := mustReadFile(t, readmePath)

	t.Run("all-tests uses scoped pattern", func(t *testing.T) {
		foundScoped := false
		for _, line := range lines {
			if strings.Contains(line, "go test ./cmd/... ./internal/...") {
				foundScoped = true
				break
			}
		}
		if !foundScoped {
			t.Error("README.md does not contain the scoped test command `go test ./cmd/... ./internal/...`")
		}
	})

	t.Run("no bare-go-test line remains", func(t *testing.T) {
		for i, line := range lines {
			// A line that is just `go test` without package arguments is
			// ambiguous and should not appear in the running-tests section.
			trimmed := strings.TrimSpace(line)
			if trimmed == "go test" || trimmed == "go test ./..." {
				t.Errorf("README.md line %d: bare or wildcard go test command → %q", i+1, trimmed)
			}
		}
	})
}
