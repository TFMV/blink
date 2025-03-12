package blink

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TFMV/blink/pkg/logger"
)

// Common development-related patterns to exclude
var defaultDevExcludePatterns = []string{
	// Version control systems
	// Git
	".git", ".gitignore", ".gitattributes", ".gitmodules", ".gitkeep",
	".git/**", "**/.git/**", "*.git~", "*.swp", "*.swo",

	// Mercurial
	".hg", ".hgignore", ".hgsub", ".hgsubstate", ".hgtags",

	// SVN
	".svn", "**/.svn/**",

	// Bazaar
	".bzr", "**/.bzr/**",

	// Python
	"__pycache__", "*.pyc", "*.pyo", "*.pyd", ".Python",
	"venv", "env", ".venv", ".env", "virtualenv", ".virtualenv",
	"ENV", "env.bak", "venv.bak", "*.egg-info", "*.egg",
	".installed.cfg", "pip-log.txt", "pip-delete-this-directory.txt",
	".coverage", ".coverage.*", ".pytest_cache", ".tox", ".nox",
	".hypothesis", "htmlcov", "site-packages", "lib/python*",

	// Node.js / JavaScript
	"node_modules", "npm-debug.log", "yarn-debug.log", "yarn-error.log",
	".npm", ".yarn", ".pnp", ".pnp.js", ".cache", ".parcel-cache",
	"bower_components", ".bower-cache", ".bower-registry", ".bower-tmp",
	".eslintcache", ".stylelintcache", ".next", ".nuxt", ".output",
	"dist", "build", "out", ".DS_Store",

	// Java / Gradle / Maven
	"*.class", "*.jar", "*.war", "*.ear", "*.log",
	".gradle", "build/", ".mvn", "target/", "*.iml",

	// Ruby
	".bundle", "vendor/bundle", ".gem", "Gemfile.lock",

	// Rust
	"target/", "Cargo.lock", "**/*.rs.bk",

	// Go
	"go.sum", "vendor/", "bin/", "pkg/",

	// .NET / C#
	"bin/", "obj/", "*.suo", "*.user", "*.userosscache", "*.dbmdl",

	// Swift / Xcode
	".build/", "Packages/", "xcuserdata/", "*.xcscmblueprint",
	"*.xccheckout", "DerivedData/", "*.moved-aside",

	// Docker
	".docker", "docker-compose.override.yml",

	// IDE and editor files
	".idea", ".vscode", ".vs", "*.swp", "*.swo", "*~",
	".project", ".classpath", ".settings", "*.sublime-*",

	// Logs and databases
	"*.log", "*.sqlite", "*.sqlite3", "*.db",

	// OS specific files
	".DS_Store", ".AppleDouble", ".LSOverride", "Thumbs.db", "ehthumbs.db",
	"Desktop.ini", "$RECYCLE.BIN", "*.lnk",

	// Temporary files
	"tmp/", "temp/", "*.tmp", "*.temp",

	// Documentation
	"docs/_build", "docs/site", "site/", ".mkdocs",

	// Package managers
	"package-lock.json", "yarn.lock", "composer.lock", "poetry.lock",

	// Test coverage
	"coverage/", ".nyc_output", "coverage.xml", "nosetests.xml",

	// Linters and formatters
	".eslintcache", ".stylelintcache", ".mypy_cache", ".ruff_cache",
	".dmypy.json", "dmypy.json", ".pyre", ".pytype",

	// Jupyter Notebooks
	".ipynb_checkpoints",

	// Dependency directories for various languages
	"lib/", "libs/", "lib64/", "vendor/", "third_party/",

	// Build artifacts
	"dist/", "build/", "out/", "bin/", "obj/", "target/",

	// Caches
	".cache/", ".caches/", "**/__pycache__/**",
}

// Common paths to exclude entirely (these are checked against the full path)
var defaultDevExcludePaths = []string{
	// Python virtual environments
	"venv", "env", ".venv", ".env",
	"python/venv", "python/.venv", "python/env", "python/.env",
	"**/venv/**", "**/.venv/**", "**/env/**", "**/.env/**",
	"**/site-packages/**", "**/dist-packages/**",

	// Node.js
	"**/node_modules/**",

	// Build directories
	"**/build/**", "**/dist/**", "**/target/**",

	// Version control
	"**/.git/**", "**/.svn/**", "**/.hg/**",

	// IDE
	"**/.idea/**", "**/.vscode/**",
}

// IsDevProject checks if the given path is a development project
func IsDevProject(path string) bool {
	// Check for common development project indicators
	indicators := []string{
		// Version control
		".git", ".hg", ".svn", ".bzr",

		// Config files
		"package.json", "composer.json", "Gemfile", "requirements.txt",
		"Cargo.toml", "go.mod", "pom.xml", "build.gradle", "setup.py",
		"pyproject.toml", "Pipfile", "*.csproj", "*.sln", "Makefile",

		// Common project directories
		"src", "lib", "test", "tests", "spec", "docs",
	}

	for _, indicator := range indicators {
		matches, err := filepath.Glob(filepath.Join(path, indicator))
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	return false
}

// GetDevExcludePatterns returns a comma-separated string of patterns to exclude for development projects
func GetDevExcludePatterns() string {
	return strings.Join(defaultDevExcludePatterns, ",")
}

// ShouldExcludePath checks if a path should be excluded based on full path exclusions
func ShouldExcludePath(path string) bool {
	// Normalize path
	normalizedPath := filepath.ToSlash(path)

	// Debug the path being checked
	logger.Debugf("Checking path for exclusion: %s", normalizedPath)

	// Special case for the specific path format we're seeing in logs
	if strings.Contains(normalizedPath, "python/blink/venv/") {
		logger.Debugf("MATCH: Excluding specific Python venv path: %s", path)
		return true
	}

	// Check for common virtual environment paths
	venvPaths := []string{"/venv/", "/.venv/", "/env/", "/.env/", "/site-packages/", "/dist-packages/"}
	for _, venv := range venvPaths {
		if strings.Contains(normalizedPath, venv) {
			logger.Debugf("MATCH: Excluding Python virtual environment path: %s (matched: %s)", path, venv)
			return true
		}
	}

	// Additional debug for Python venv paths
	if strings.Contains(normalizedPath, "venv") || strings.Contains(normalizedPath, "site-packages") {
		logger.Debugf("DEBUG: Path contains 'venv' or 'site-packages' but didn't match exact patterns: %s", normalizedPath)
	}

	// Check against excluded paths with glob pattern support
	for _, excludePath := range defaultDevExcludePaths {
		logger.Debugf("DEBUG: Checking against exclude path: %s", excludePath)

		// Handle ** pattern (recursive matching)
		if strings.Contains(excludePath, "**") {
			// Convert ** to a more specific check
			parts := strings.Split(excludePath, "**")
			allPartsMatch := true

			// Check if all parts of the pattern exist in the path in the correct order
			lastIndex := 0
			for _, part := range parts {
				if part == "" {
					continue
				}

				index := strings.Index(normalizedPath[lastIndex:], part)
				if index == -1 {
					allPartsMatch = false
					logger.Debugf("DEBUG: Part '%s' of pattern '%s' not found in path: %s", part, excludePath, normalizedPath[lastIndex:])
					break
				}
				lastIndex += index + len(part)
			}

			if allPartsMatch {
				logger.Debugf("MATCH: Excluding path by ** pattern: %s (pattern: %s)", path, excludePath)
				return true
			}
			continue
		}

		// Try direct glob matching
		matched, err := filepath.Match(excludePath, normalizedPath)
		if err != nil {
			logger.Debugf("DEBUG: Error matching pattern '%s' against path '%s': %v", excludePath, normalizedPath, err)
		} else if matched {
			logger.Debugf("MATCH: Excluding path by glob match: %s (pattern: %s)", path, excludePath)
			return true
		}

		// Try matching against path segments
		pathParts := strings.Split(normalizedPath, "/")
		for _, part := range pathParts {
			if part == "" {
				continue
			}
			matched, err := filepath.Match(excludePath, part)
			if err != nil {
				logger.Debugf("DEBUG: Error matching pattern '%s' against segment '%s': %v", excludePath, part, err)
			} else if matched {
				logger.Debugf("MATCH: Excluding path by segment match: %s (pattern: %s, segment: %s)", path, excludePath, part)
				return true
			}
		}

		// Check if the path contains any of the excluded paths (substring match)
		if !strings.Contains(excludePath, "*") && strings.Contains(normalizedPath, excludePath) {
			logger.Debugf("MATCH: Excluding path by substring match: %s (matched: %s)", path, excludePath)
			return true
		}
	}

	// Log if path contains 'python' and 'venv' but wasn't matched
	if strings.Contains(normalizedPath, "python") && strings.Contains(normalizedPath, "venv") {
		logger.Debugf("NOT MATCHED: Python venv path not excluded: %s", normalizedPath)
	}

	// Check for node_modules
	if strings.Contains(normalizedPath, "/node_modules/") {
		logger.Debugf("MATCH: Excluding Node.js modules path: %s", path)
		return true
	}

	// Check for version control directories
	vcsPaths := []string{"/.git/", "/.svn/", "/.hg/"}
	for _, vcs := range vcsPaths {
		if strings.Contains(normalizedPath, vcs) {
			logger.Debugf("MATCH: Excluding version control path: %s (matched: %s)", path, vcs)
			return true
		}
	}

	// Check for IDE directories
	idePaths := []string{"/.idea/", "/.vscode/"}
	for _, ide := range idePaths {
		if strings.Contains(normalizedPath, ide) {
			logger.Debugf("MATCH: Excluding IDE path: %s (matched: %s)", path, ide)
			return true
		}
	}

	// Check for cache directories
	cachePaths := []string{"/__pycache__/", "/.pytest_cache/", "/.mypy_cache/", "/.ruff_cache/"}
	for _, cache := range cachePaths {
		if strings.Contains(normalizedPath, cache) {
			logger.Debugf("MATCH: Excluding cache path: %s (matched: %s)", path, cache)
			return true
		}
	}

	// Check for build directories
	buildPaths := []string{"/build/", "/dist/", "/target/"}
	for _, build := range buildPaths {
		if strings.Contains(normalizedPath, build) {
			logger.Debugf("MATCH: Excluding build path: %s (matched: %s)", path, build)
			return true
		}
	}

	// Check for common file extensions to exclude
	excludeExts := []string{".pyc", ".pyo", ".pyd", ".so", ".dll", ".exe", ".obj", ".o"}
	for _, ext := range excludeExts {
		if strings.HasSuffix(normalizedPath, ext) {
			logger.Debugf("MATCH: Excluding file by extension: %s (matched: %s)", path, ext)
			return true
		}
	}

	return false
}

// ApplyDevFilter applies development-related exclusion patterns to the given filter
func ApplyDevFilter(filter *EventFilter, path string) {
	// Always apply dev filtering, but log if it's a recognized project
	if IsDevProject(path) {
		logger.Info("Development project detected, applying comprehensive filtering")
	} else {
		logger.Debug("Applying standard development filtering")
	}

	// Get the current exclude patterns
	currentExcludePatterns := filter.GetExcludePatterns()

	// Add development-related patterns
	devPatterns := GetDevExcludePatterns()

	// Combine patterns
	var newExcludePatterns string
	if currentExcludePatterns != "" {
		newExcludePatterns = currentExcludePatterns + "," + devPatterns
	} else {
		newExcludePatterns = devPatterns
	}

	// Set the new exclude patterns
	filter.SetExcludePatterns(newExcludePatterns)
	logger.Debugf("Set exclude patterns: %s", newExcludePatterns)

	// Add a custom filter function to check full paths
	filter.AddCustomFilter(func(path string, isDir bool) bool {
		logger.Debugf("Custom filter checking path: %s (isDir: %v)", path, isDir)

		// Check if the path should be excluded
		if strings.Contains(path, "python/blink/venv/") {
			logger.Debugf("Custom filter excluding Python venv path: %s", path)
			return false
		}

		// Check for site-packages
		if strings.Contains(path, "site-packages") {
			logger.Debugf("Custom filter excluding site-packages path: %s", path)
			return false
		}

		// Check for other common patterns
		if ShouldExcludePath(path) {
			logger.Debugf("Custom filter excluding path: %s", path)
			return false
		}

		logger.Debugf("Custom filter including path: %s", path)
		return true
	})

	logger.Infof("Development filtering applied")
}

// Legacy functions for backward compatibility

// IsGitRepository checks if the given path is a Git repository
func IsGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetGitExcludePatterns returns a comma-separated string of patterns to exclude for Git repositories
func GetGitExcludePatterns() string {
	// Extract Git-specific patterns from the unified list
	gitPatterns := []string{}
	for _, pattern := range defaultDevExcludePatterns {
		if strings.Contains(pattern, ".git") || strings.HasSuffix(pattern, ".swp") || strings.HasSuffix(pattern, ".swo") {
			gitPatterns = append(gitPatterns, pattern)
		}
	}
	return strings.Join(gitPatterns, ",")
}

// ApplyGitFilter applies Git-related exclusion patterns to the given filter
// Kept for backward compatibility
func ApplyGitFilter(filter *EventFilter, path string) {
	logger.Info("Git filtering is deprecated, using comprehensive development filtering instead")
	ApplyDevFilter(filter, path)
}

// IsPythonProject checks if the given path contains Python files
func IsPythonProject(path string) bool {
	// Check for common Python project indicators
	indicators := []string{
		"*.py",
		"requirements.txt",
		"setup.py",
		"pyproject.toml",
		"Pipfile",
		"venv",
		".venv",
	}

	for _, indicator := range indicators {
		matches, err := filepath.Glob(filepath.Join(path, indicator))
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	return false
}

// GetPythonExcludePatterns returns a comma-separated string of patterns to exclude for Python projects
func GetPythonExcludePatterns() string {
	// Extract Python-specific patterns from the unified list
	pythonPatterns := []string{}
	for _, pattern := range defaultDevExcludePatterns {
		if strings.Contains(pattern, "py") || strings.Contains(pattern, "venv") ||
			strings.Contains(pattern, "__pycache__") || strings.Contains(pattern, ".env") {
			pythonPatterns = append(pythonPatterns, pattern)
		}
	}
	return strings.Join(pythonPatterns, ",")
}

// ApplyPythonFilter applies Python-related exclusion patterns to the given filter
// Kept for backward compatibility
func ApplyPythonFilter(filter *EventFilter, path string) {
	logger.Info("Python filtering is deprecated, using comprehensive development filtering instead")
	ApplyDevFilter(filter, path)
}
