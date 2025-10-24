package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to create a mock executor with default settings
func createMockExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		Output: []byte("mocked helm output"),
		Error:  nil,
	}
}

// Helper function to create a mock executor with custom behavior
func createMockExecutorWithBehavior(behaviorFunc func() error) *MockCommandExecutor {
	mockExecutor := createMockExecutor()
	mockExecutor.BehaviorOnRun = behaviorFunc
	return mockExecutor
}

// Helper function to create a context for tests
func createTestContext() context.Context {
	return context.Background()
}

// Common validation helper for string slices
func assertStringSlicesEqual(t *testing.T, expected, actual []string, message string) {
	assert.Equal(t, len(expected), len(actual), "Length mismatch: %s", message)
	for i, expectedVal := range expected {
		if i < len(actual) {
			assert.Equal(t, expectedVal, actual[i], "Mismatch at index %d: %s", i, message)
		}
	}
}

// Helper to assert command execution
func assertCommandExecution(t *testing.T, mockExecutor *MockCommandExecutor, expectedCommand string) {
	actualCommand := mockExecutor.GetFullCommand()
	assert.Equal(t, expectedCommand, actualCommand)
}

// Helper to create a default test chart for chart rendering tests
func createTestChart() ChartRenderParams {
	return ChartRenderParams{
		Env:            "development",
		ChartName:      "test-chart",
		RepoURL:        "https://example.com/charts",
		BaseValuesFile: "values.yaml",
		ValuesOverride: "override.yaml",
		ChartVersion:   "1.0.0",
	}
}

// Helper function to assert chart fields match
func assertChartFieldsMatch(t *testing.T, expected, actual ChartRenderParams) {
	assert.Equal(t, expected.ChartName, actual.ChartName)
	assert.Equal(t, expected.RepoURL, actual.RepoURL)
	assert.Equal(t, expected.BaseValuesFile, actual.BaseValuesFile)
	assert.Equal(t, expected.ValuesOverride, actual.ValuesOverride)
	assert.Equal(t, expected.ChartVersion, actual.ChartVersion)
}

// Helper function to create an image extraction engine
func createImageExtractionEngine() *ImageExtractionEngine {
	return &ImageExtractionEngine{
		inputChan:  make(chan ManifestValidationResult),
		outputChan: make(chan ImageExtractionResult),
		context:    createTestContext(),
	}
}

// Helper function to create a temp manifest file
func createTempManifestFile(t *testing.T, tempDir, filename, content string) string {
	manifestPath := filepath.Join(tempDir, filename)
	err := os.MkdirAll(filepath.Dir(manifestPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	err = os.WriteFile(manifestPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp manifest file: %v", err)
	}
	return manifestPath
}

// Helper function to collect image extraction results
func collectImageExtractionResults(engine *ImageExtractionEngine) []ImageExtractionResult {
	results := make([]ImageExtractionResult, 0)
	for extractionResult := range engine.outputChan {
		results = append(results, extractionResult)
	}
	return results
}

// Helper function to extract image names from results
func extractImageNames(results []ImageExtractionResult) []string {
	var images []string
	for _, result := range results {
			images = append(images, result.Image)
	}
	return images
}

// Helper function to assert image set matches
func assertImageSetMatches(t *testing.T, expected map[string]bool, actual []string, testName string) {
	if len(actual) != len(expected) {
		t.Errorf("Expected %d images, got %d for %s", len(expected), len(actual), testName)
	}
	for _, img := range actual {
		if !expected[img] {
			t.Errorf("Unexpected image %s found for %s", img, testName)
		}
	}
}

// Helper function to assert string slices match exactly
func assertStringSlicesMatch(t *testing.T, expected, actual []string, testName string) {
	if len(actual) != len(expected) {
		t.Errorf("Expected %d items, got %d for %s", len(expected), len(actual), testName)
		return
	}
	for i, expectedItem := range expected {
		if i < len(actual) && actual[i] != expectedItem {
			t.Errorf("Expected item %s at index %d, got %s for %s", expectedItem, i, actual[i], testName)
		}
	}
}

// Helper function to create a manifest validation engine
func createManifestValidationEngine(mockExecutor *MockCommandExecutor) *ManifestValidationEngine {
	return &ManifestValidationEngine{
		inputChan:  make(chan RenderResult),
		resultChan: make(chan ManifestValidationResult),
		context:    createTestContext(),
		executor:   mockExecutor,
		errorChan:  make(chan ErrorResult),
	}
}

// Helper function to send render result to manifest validation engine
func sendRenderResultToEngine(engine *ManifestValidationEngine, manifestPath string) {
	go func() {
		engine.inputChan <- RenderResult{
			ManifestPath: manifestPath,
		}
	}()
}

// Helper function to create default mock executor for manifest validation
func createManifestValidationMockExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		Output: []byte("mocked kubeconform output"),
		Error:  nil,
	}
}