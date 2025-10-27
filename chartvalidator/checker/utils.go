package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// logEngine prints formatted log messages with color coding based on level
func logEngine(level, engineName string, workerId int, message string) {
	var color string
	switch strings.ToUpper(level) {
	case "ERROR":
		color = colorRed
	case "WARNING":
		color = colorYellow
	case "DEBUG":
		color = colorCyan
	default:
		color = colorReset
	}

	// Split message into lines if it contains newlines
	lines := strings.Split(message, "\n")
	
	// Print first line with full prefix and color
	fmt.Printf("%s[%s]\t[%s Worker %d]\t%s%s\n", color, level, engineName, workerId, lines[0], colorReset)
	
	// Print additional lines with empty columns for alignment
	for i := 1; i < len(lines); i++ {
		fmt.Printf("\t\t%s\n", lines[i])
	}
}

func logEngineDebug(engineName string, workerId int, message string) {
	if !verboseLogging {
		return
	}
	logEngine("DEBUG", engineName, workerId, message)
}

func logEngineWarning(engineName string, workerId int, message string) {
	logEngine("WARNING", engineName, workerId, message)
}

func logEngineError(engineName string, workerId int, message string) {
	logEngine("ERROR", engineName, workerId, message)
}

// getJobCount returns the number of parallel jobs to run
func getJobCount() int {
	if s := os.Getenv("KUBECONFORM_JOBS"); strings.TrimSpace(s) != "" {
		if n, err := parseInt(s); err == nil && n > 0 {
			return n
		}
	}
	n := runtime.NumCPU()
	if n <= 0 {
		n = 4
	}
	return n
}

// parseInt parses a string to integer, returning error if invalid
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}

// recreateOutputDir removes and recreates the output directory
func recreateOutputDir(outputDir string) error {
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to remove output directory: %w", err)
	}
	
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	return nil
}

// walkFiles returns all files under root that pass the filter
func walkFiles(root string, filter func(string, fs.DirEntry) bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filter(p, d) {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

// removeDuplicates removes duplicate strings from a slice while preserving order
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// findYAMLFiles discovers all YAML files in a directory recursively
func findYAMLFiles(dir string) ([]string, error) {
	return walkFiles(dir, func(path string, d fs.DirEntry) bool {
		name := strings.ToLower(d.Name())
		return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
	})
}