package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper function to create a Docker validation engine
func createDockerValidationEngine(mockExecutor *MockCommandExecutor) *DockerImageValidationEngine {
	return &DockerImageValidationEngine{
		inputChan:  make(chan ImageExtractionResult),
		outputChan: make(chan DockerImageValidationResult),
		executor:   mockExecutor,
		context:    createTestContext(),
		cache:      make(map[string]DockerImageValidationResult),
		pending:    make(map[string]*sync.WaitGroup),
		name:       "DockerImageValidationEngine",
	}
}

// Helper function to create test images slice
func createTestImages() []string {
	return []string{
		"nginx:1.20",
		"redis:6.2",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
		"redis:6.2",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
		"redis:6.2",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
		"redis:6.2",
		"nginx:1.20",
		"nginx:1.21",
		"nginx:1.21",
		"nginx:1.21",
		"redis:6.2",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
		"nginx:1.20",
	}
}

// Helper function to send images to engine
func sendImagesToEngine(engine *DockerImageValidationEngine, images []string) {
	go func() {
		for _, img := range images {
			engine.inputChan <- ImageExtractionResult{
				Image: img,
			}
		}
	}()
}

// Helper function to collect results from engine
func collectResults(engine *DockerImageValidationEngine, count int) map[string]DockerImageValidationResult {
	resultStore := make(map[string]DockerImageValidationResult)
	for i := 0; i < count; i++ {
		result := <-engine.outputChan
		resultStore[result.Image] = result
	}
	return resultStore
}

// Helper function to create test files in directory
func createTestFiles(t *testing.T, tempDir string, files []string) {
	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}
}

// Helper function to create JSON file with content
func createJSONFile(t *testing.T, filePath string, content []string) {
	jsonData, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}
}

func TestDockerImageValidationEngine(t *testing.T) {
	mockExecutor := createMockExecutor()
	engine := createDockerValidationEngine(mockExecutor)
	engine.Start(1)

	img := "nginx:1.20"
	go func(s string) {
		engine.inputChan <- ImageExtractionResult{
			Image: s,
		}
	}(img)

	result := <-engine.outputChan
	if result.Image != img {
		t.Errorf("Expected image %s, got %s", img, result.Image)
	}
	if !result.Exists {
		t.Errorf("Expected image %s to exist", img)
	}

	assertCommandExecution(t, mockExecutor, "docker manifest inspect nginx:1.20")
	engine.context.Done()
}

func TestDockerImageValidationCache(t *testing.T) {
	mockExecutor := createMockExecutorWithBehavior(func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	engine := createDockerValidationEngine(mockExecutor)
	engine.Start(2)

	images := createTestImages()
	sendImagesToEngine(engine, images)
	resultStore := collectResults(engine, len(images))

	if len(resultStore) != 3 {
		t.Errorf("Expected 3 unique results, got %d", len(resultStore))
	}

	engine.context.Done()
}


// TestFindJSONFiles tests finding JSON files in a directory
func TestFindJSONFiles(t *testing.T) {
	tempDir := t.TempDir()

	jsonFiles := []string{
		"images1.json",
		"images2.json",
		"subdir/nested.json",
	}

	nonJSONFiles := []string{
		"config.yaml",
		"readme.txt",
		"data.xml",
	}

	allFiles := append(jsonFiles, nonJSONFiles...)
	createTestFiles(t, tempDir, allFiles)

	foundFiles, err := findJSONFiles(tempDir)
	if err != nil {
		t.Fatalf("findJSONFiles failed: %v", err)
	}

	if len(foundFiles) != len(jsonFiles) {
		t.Errorf("Expected %d JSON files, found %d", len(jsonFiles), len(foundFiles))
	}

	// Convert to relative paths for comparison
	foundSet := make(map[string]bool)
	for _, file := range foundFiles {
		rel, err := filepath.Rel(tempDir, file)
		if err != nil {
			t.Fatalf("Failed to get relative path: %v", err)
		}
		foundSet[rel] = true
	}

	// Check all expected JSON files are found
	for _, expected := range jsonFiles {
		if !foundSet[expected] {
			t.Errorf("Expected JSON file %s not found", expected)
		}
	}
}

// TestExtractImagesFromJSONFile tests extracting images from a single JSON file
func TestExtractImagesFromJSONFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		jsonContent    []string
		expectedImages []string
		expectError    bool
	}{
		{
			name:           "valid JSON with images",
			jsonContent:    []string{"nginx:1.20", "redis:6.2", "postgres:13"},
			expectedImages: []string{"nginx:1.20", "redis:6.2", "postgres:13"},
			expectError:    false,
		},
		{
			name:           "empty JSON array",
			jsonContent:    []string{},
			expectedImages: []string{},
			expectError:    false,
		},
		{
			name:           "single image",
			jsonContent:    []string{"alpine:latest"},
			expectedImages: []string{"alpine:latest"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonFile := filepath.Join(tempDir, tt.name+".json")
			createJSONFile(t, jsonFile, tt.jsonContent)

			images, err := extractImagesFromJSONFile(jsonFile)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				assertStringSlicesEqual(t, tt.expectedImages, images, "extracted images")
			}
		})
	}
}

// TestExtractImagesFromJSONFileInvalidJSON tests handling of invalid JSON
func TestExtractImagesFromJSONFileInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	invalidJSON := `{"invalid": "json", "not": ["an", "array"]}`
	jsonFile := filepath.Join(tempDir, "invalid.json")
	err := os.WriteFile(jsonFile, []byte(invalidJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	_, err = extractImagesFromJSONFile(jsonFile)
	if err == nil {
		t.Errorf("Expected error for invalid JSON, but got none")
	}
}

// TestExtractAllImagesFromJSONFiles tests extracting images from multiple JSON files
func TestExtractAllImagesFromJSONFiles(t *testing.T) {
	tempDir := t.TempDir()

	testFiles := map[string][]string{
		"file1.json": {"nginx:1.20", "redis:6.2"},
		"file2.json": {"postgres:13", "alpine:latest"},
		"file3.json": {"node:16", "python:3.9"},
	}

	var allPaths []string
	var expectedImages []string

	for filename, images := range testFiles {
		jsonFile := filepath.Join(tempDir, filename)
		createJSONFile(t, jsonFile, images)
		allPaths = append(allPaths, jsonFile)
		expectedImages = append(expectedImages, images...)
	}

	allImages, err := extractAllImagesFromJSONFiles(allPaths)
	if err != nil {
		t.Fatalf("extractAllImagesFromJSONFiles failed: %v", err)
	}

	if len(allImages) != len(expectedImages) {
		t.Errorf("Expected %d total images, got %d", len(expectedImages), len(allImages))
	}

	// Check all expected images are present (order might differ)
	imageSet := make(map[string]bool)
	for _, img := range allImages {
		imageSet[img] = true
	}

	for _, expected := range expectedImages {
		if !imageSet[expected] {
			t.Errorf("Expected image %s not found in results", expected)
		}
	}
}

// TestDeduplicateImages tests image deduplication
func TestDeduplicateImages(t *testing.T) {
	tests := []struct {
		name           string
		input          []string
		expectedUnique []string
	}{
		{
			name:           "no duplicates",
			input:          []string{"nginx:1.20", "redis:6.2", "postgres:13"},
			expectedUnique: []string{"nginx:1.20", "postgres:13", "redis:6.2"}, // sorted
		},
		{
			name:           "with duplicates",
			input:          []string{"nginx:1.20", "redis:6.2", "nginx:1.20", "postgres:13", "redis:6.2"},
			expectedUnique: []string{"nginx:1.20", "postgres:13", "redis:6.2"}, // sorted and deduplicated
		},
		{
			name:           "empty input",
			input:          []string{},
			expectedUnique: []string{},
		},
		{
			name:           "with empty strings",
			input:          []string{"nginx:1.20", "", "redis:6.2", ""},
			expectedUnique: []string{"nginx:1.20", "redis:6.2"}, // empty strings filtered out
		},
		{
			name:           "all same",
			input:          []string{"nginx:1.20", "nginx:1.20", "nginx:1.20"},
			expectedUnique: []string{"nginx:1.20"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateImages(tt.input)
			assertStringSlicesEqual(t, tt.expectedUnique, result, "deduplicated images")
		})
	}
}

// TestCreateDockerManifestInspectCommand tests the docker command creation
func TestCreateDockerManifestInspectCommand(t *testing.T) {
	tests := []struct {
		name         string
		image        string
		expectedArgs []string
	}{
		{
			name:         "simple image",
			image:        "nginx:1.20",
			expectedArgs: []string{"manifest", "inspect", "nginx:1.20"},
		},
		{
			name:         "image with registry",
			image:        "registry.example.com/my-app:v1.0",
			expectedArgs: []string{"manifest", "inspect", "registry.example.com/my-app:v1.0"},
		},
		{
			name:         "image with digest",
			image:        "nginx@sha256:abc123",
			expectedArgs: []string{"manifest", "inspect", "nginx@sha256:abc123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createDockerManifestInspectCommand(tt.image)

			if filepath.Base(cmd.Path) != "docker" {
				t.Errorf("Expected docker command, got %s", cmd.Path)
			}

			// cmd.Args[0] is the program name, cmd.Args[1:] are the actual arguments
			actualArgs := cmd.Args[1:]
			assertStringSlicesEqual(t, tt.expectedArgs, actualArgs, "docker command arguments")
		})
	}
}

// TestValidateSingleDockerImage tests the validation logic (without actually calling docker)
func TestValidateSingleDockerImage(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		expectedImage string
	}{
		{
			name:          "valid image name",
			image:         "nginx:1.20",
			expectedImage: "nginx:1.20",
		},
		{
			name:          "image with registry",
			image:         "gcr.io/my-project/my-app:latest",
			expectedImage: "gcr.io/my-project/my-app:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createDockerManifestInspectCommand(tt.image)
			assert.NotNil(t, cmd, "command should not be nil")
			assert.Equal(t, cmd.Args[0], "docker", "command should be docker")
			assert.Equal(t, cmd.Args[1], "manifest", "command should be manifest")
			assert.Equal(t, cmd.Args[2], "inspect", "command should be inspect")
			if cmd.Args[len(cmd.Args)-1] != tt.expectedImage {
				t.Errorf("Expected command to include image %s, got args %v", tt.expectedImage, cmd.Args)
			}
		})
	}
}

func TestDockerValidationError(t *testing.T) {
	mockExecutor := createMockExecutorWithBehavior(func() error {
		return fmt.Errorf("mocked docker error")
	})

	engine := createDockerValidationEngine(mockExecutor)
	engine.Start(1)

	img := "nonexistent:image"
	go func(s string) {
		engine.inputChan <- ImageExtractionResult{
			Image: s,
		}
	}(img)

	result := <-engine.outputChan
	assert.Equal(t, result.Image, img)
	assert.NotNil(t, result.Error)
	assertCommandExecution(t, mockExecutor, "docker manifest inspect nonexistent:image")
	engine.context.Done()
}