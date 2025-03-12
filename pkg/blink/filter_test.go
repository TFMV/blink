package blink

import (
	"testing"
)

func TestShouldExcludePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Python virtual environment paths
		{"Python venv path", "python/blink/venv/lib/python3.13/site-packages/file.py", true},
		{"Python site-packages path", "/usr/local/lib/python3.13/site-packages/file.py", true},
		{"Python dist-packages path", "/usr/lib/python3/dist-packages/file.py", true},
		{"Python env path", "project/env/lib/python3.13/file.py", true},
		{"Python .venv path", "project/.venv/bin/python", true},

		// Node.js paths
		{"Node modules path", "project/node_modules/lodash/index.js", true},
		{"Nested node modules", "project/subdir/node_modules/react/index.js", true},

		// Version control paths
		{"Git directory", "project/.git/HEAD", true},
		{"Git subdirectory", "project/.git/objects/pack/file.idx", true},
		{"SVN directory", "project/.svn/entries", true},
		{"Mercurial directory", "project/.hg/dirstate", true},

		// IDE paths
		{"VSCode directory", "project/.vscode/settings.json", true},
		{"IntelliJ directory", "project/.idea/workspace.xml", true},

		// Cache directories
		{"Python cache", "project/__pycache__/module.pyc", true},
		{"Pytest cache", "project/.pytest_cache/v/cache/nodeids", true},
		{"MyPy cache", "project/.mypy_cache/3.9/module.meta.json", true},
		{"Ruff cache", "project/.ruff_cache/file.py", true},

		// Build directories
		{"Build directory", "project/build/output.o", true},
		{"Dist directory", "project/dist/bundle.js", true},
		{"Target directory", "project/target/classes/Main.class", true},

		// File extensions
		{"Python compiled file", "project/module.pyc", true},
		{"Python optimized file", "project/module.pyo", true},
		{"Shared object file", "project/library.so", true},
		{"DLL file", "project/library.dll", true},
		{"Executable file", "project/program.exe", true},
		{"Object file", "project/module.o", true},

		// Glob patterns
		{"Double star pattern", "project/subdir/node_modules/package/file.js", true},
		{"Single star pattern", "project/.git/config", true},

		// Files that should not be excluded
		{"Regular Python file", "project/module.py", false},
		{"Regular JavaScript file", "project/script.js", false},
		{"Regular HTML file", "project/index.html", false},
		{"Regular CSS file", "project/styles.css", false},
		{"Regular Go file", "project/main.go", false},
		{"Regular Java file", "project/Main.java", false},
		{"Regular text file", "project/README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldExcludePath(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldExcludePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestEventFilterShouldIncludePath(t *testing.T) {
	tests := []struct {
		name            string
		includePatterns string
		excludePatterns string
		path            string
		expected        bool
	}{
		// Basic include patterns
		{"Include JS files", "*.js", "", "project/script.js", true},
		{"Include JS files - non-match", "*.js", "", "project/style.css", false},
		{"Include multiple patterns", "*.js,*.css,*.html", "", "project/style.css", true},

		// Basic exclude patterns
		{"Exclude node_modules", "", "node_modules", "project/node_modules/package.json", false},
		{"Exclude multiple patterns", "", "node_modules,*.tmp", "project/file.tmp", false},

		// Combined patterns
		{"Include and exclude", "*.js", "node_modules", "project/script.js", true},
		{"Include and exclude - excluded", "*.js", "node_modules", "project/node_modules/script.js", false},

		// Advanced glob patterns
		{"Double star include", "**/src/**/*.js", "", "project/src/components/Button.js", true},
		{"Double star exclude", "", "**/node_modules/**", "project/node_modules/package/index.js", false},

		// Path segment matching
		{"Path segment include", "src", "", "project/src/file.js", true},
		{"Path segment exclude", "", "test", "project/test/file_test.js", false},

		// Default behavior (no patterns)
		{"No patterns", "", "", "project/file.js", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewEventFilter()
			if tt.includePatterns != "" {
				filter.SetIncludePatterns(tt.includePatterns)
			}
			if tt.excludePatterns != "" {
				filter.SetExcludePatterns(tt.excludePatterns)
			}

			result := filter.ShouldIncludePath(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldIncludePath(%q) with include=%q, exclude=%q = %v, want %v",
					tt.path, tt.includePatterns, tt.excludePatterns, result, tt.expected)
			}
		})
	}
}

func TestApplyDevFilter(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		testPath string
		expected bool
	}{
		{"Python venv path", ".", "python/blink/venv/lib/python3.13/site-packages/file.py", false},
		{"Node modules path", ".", "project/node_modules/lodash/index.js", false},
		{"Git directory", ".", "project/.git/HEAD", false},
		{"Regular file", ".", "project/main.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewEventFilter()
			ApplyDevFilter(filter, tt.path)

			// Test if the filter would process this path
			for _, customFilter := range filter.customFilters {
				if customFilter != nil {
					result := customFilter(tt.testPath, false)
					if result != tt.expected {
						t.Errorf("Custom filter for %q = %v, want %v", tt.testPath, result, tt.expected)
					}
					break
				}
			}
		})
	}
}
