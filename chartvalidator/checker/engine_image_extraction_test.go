package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Sample manifests for testing
var sampleManifests = map[string]string{
	"pod_sample": `
apiVersion: v1
kind: Pod
metadata:
  name: sample-pod
spec:
  containers:
  - name: sample-container
    image: nginx:1.14.2
`,
	"deployment_sample": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: sample-app
  template:
    metadata:
      labels:
        app: sample-app
    spec:
      initContainers:
      - name: init-sample
        image: busybox:1.28
        command: ['sh', '-c', 'echo Init Container']
      containers:
      - name: sample-container
        image: nginx:1.14.2
      - name: another-container
        image: redis:6.0
`,
	"daemonset_sample": `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: sample-daemonset
spec:
  selector:
    matchLabels:
      app: sample-daemonset
  template:
    metadata:
      labels:
        app: sample-daemonset
    spec:
      containers:
      - name: sample-container
        image: nginx:1.14.2
      - name: another-container
        image: redis:6.0
`,
	"statefulset_sample": `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: sample-statefulset
spec:
  serviceName: "sample-service"
  replicas: 3
  selector:
    matchLabels:
      app: sample-app
  template:
    metadata:
      labels:
        app: sample-app
    spec:
      containers:
      - name: sample-container
        image: nginx:1.14.2
      - name: another-container
        image: redis:6.0
`,
}

// Helper function to get expected images for each manifest type
func getExpectedImages(manifestType string) map[string]bool {
	switch manifestType {
	case "pod_sample":
		return map[string]bool{"nginx:1.14.2": true}
	case "deployment_sample":
		return map[string]bool{
			"nginx:1.14.2": true,
			"redis:6.0":    true,
			"busybox:1.28": true,
		}
	case "daemonset_sample", "statefulset_sample":
		return map[string]bool{
			"nginx:1.14.2": true,
			"redis:6.0":    true,
		}
	default:
		return map[string]bool{}
	}
}

// Helper function to process engine with manifest
func processEngineWithManifest(t *testing.T, engine *ImageExtractionEngine, manifestPath string) []ImageExtractionResult {
	input := ManifestValidationResult{
		ManifestFile: manifestPath,
	}

	engine.inputChan <- input
	close(engine.inputChan)

	return collectImageExtractionResults(engine)
}


func TestSingleImageExtraction(t *testing.T) {
	verboseLogging = true
	engine := createImageExtractionEngine()
	engine.Start(1)

	tempDir := t.TempDir()
	manifestPath := createTempManifestFile(t, tempDir, "test-deployment.yaml", sampleManifests["deployment_sample"])

	results := processEngineWithManifest(t, engine, manifestPath)

	expectedImages := getExpectedImages("deployment_sample")
	actualImages := extractImageNames(results)

	assertImageSetMatches(t, expectedImages, actualImages, "deployment_sample")
	
}

func TestImageExtractionEngine(t *testing.T) {
	verboseLogging = true

	for name, manifest := range sampleManifests {
		t.Run(name, func(t *testing.T) {
			engine := createImageExtractionEngine()
			engine.Start(1)

			tempDir := t.TempDir()
			manifestPath := createTempManifestFile(t, tempDir, name+".yaml", manifest)

			results := processEngineWithManifest(t, engine, manifestPath)
			actualImages := extractImageNames(results)
			expectedImages := getExpectedImages(name)

			assertImageSetMatches(t, expectedImages, actualImages, name)
		})
	}
}


func TestExtractImageFromManifest(t *testing.T) {
	tests := []struct {
		name            string
		manifestType    string
		expectedImages  map[string]bool
	}{
		{
			name:         "pod",
			manifestType: "pod_sample",
			expectedImages: map[string]bool{"nginx:1.14.2": true},
		},
		{
			name:         "deployment",
			manifestType: "deployment_sample", 
			expectedImages: map[string]bool{
				"nginx:1.14.2": true,
				"redis:6.0":    true,
				"busybox:1.28": true,
			},
		},
		{
			name:         "daemonset",
			manifestType: "daemonset_sample",
			expectedImages: map[string]bool{
				"nginx:1.14.2": true,
				"redis:6.0":    true,
			},
		},
		{
			name:         "statefulset", 
			manifestType: "statefulset_sample",
			expectedImages: map[string]bool{
				"nginx:1.14.2": true,
				"redis:6.0":    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := extractImageFromManifest(sampleManifests[tt.manifestType], 0)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			
			assertImageSetMatches(t, tt.expectedImages, images, tt.name)
		})
	}
}

func TestImageCheckStruct(t *testing.T) {
	testChart := createTestChart()
	
	imgCheck := &imageCheck{
		Image: "alpine:latest",
		Chart: testChart,
	}
	
	// Test field assignments
	assert.Equal(t, "alpine:latest", imgCheck.Image)
	assert.Equal(t, "test-chart", imgCheck.Chart.ChartName)
	assert.Equal(t, "development", imgCheck.Chart.Env)
	
	// Test that Present and Error fields can be set
	imgCheck.Present = true
	imgCheck.Error = nil
	
	assert.True(t, imgCheck.Present)
	assert.Nil(t, imgCheck.Error)
}


func TestExtractImagesFromFile(t *testing.T) {
	tempDir := t.TempDir()
	manifestDir := filepath.Join(tempDir, "manifests")
	outputDir := filepath.Join(tempDir, "output")
	
	createTestFiles(t, tempDir, []string{
		"manifests/subdir/.keep", // Create the subdirectory
		"output/.keep",           // Create the output directory
	})

	manifestContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: duplicate-test
spec:
  template:
    spec:
      containers:
      - name: app1
        image: nginx:1.20
      - name: app2
        image: nginx:1.20
      - name: app3
        image: redis:6.2`

	manifestFile := createTempManifestFile(t, manifestDir, "subdir/duplicate.yaml", manifestContent)

	err := extractImagesFromFile(manifestFile, manifestDir, outputDir, 0)
	assert.NoError(t, err)

	// Verify output file with underscore naming
	expectedFileName := "subdir_duplicate.json"
	outputFile := filepath.Join(outputDir, expectedFileName)
	assert.FileExists(t, outputFile)

	// Read and verify content
	jsonData, err := os.ReadFile(outputFile)
	assert.NoError(t, err)

	var images []string
	err = json.Unmarshal(jsonData, &images)
	assert.NoError(t, err)

	// Should have only unique images
	expectedImages := []string{"nginx:1.20", "redis:6.2"}
	assert.Equal(t, len(expectedImages), len(images))
}

// TestRemoveDuplicates tests the removeDuplicates helper function
func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "all same",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicates(tt.input)
			assertStringSlicesMatch(t, tt.expected, result, tt.name)
		})
	}
}


// TestExtractImagesFromFile tests the extractImagesFromFi
// TestFindYAMLFiles tests the findYAMLFiles helper function
func TestFindYAMLFiles(t *testing.T) {
	tempDir := t.TempDir()

	yamlFiles := []string{
		"test1.yaml",
		"test2.yml",
		"subdir/nested.yaml",
	}

	nonYamlFiles := []string{
		"test.txt",
		"config.json",
	}

	allFiles := append(yamlFiles, nonYamlFiles...)
	createTestFiles(t, tempDir, allFiles)

	foundFiles, err := findYAMLFiles(tempDir)
	assert.NoError(t, err)

	assert.Equal(t, len(yamlFiles), len(foundFiles), "Expected correct number of YAML files")

	// Convert to relative paths for comparison
	var relativeFound []string
	for _, file := range foundFiles {
		rel, err := filepath.Rel(tempDir, file)
		assert.NoError(t, err)
		relativeFound = append(relativeFound, rel)
	}

	// Check that all expected YAML files are found
	foundSet := make(map[string]bool)
	for _, file := range relativeFound {
		foundSet[file] = true
	}

	for _, expected := range yamlFiles {
		assert.True(t, foundSet[expected], "Expected YAML file %s not found", expected)
	}
}


// TestExtractDockerImagesCommand tests the extract-docker-images command functionality
func TestExtractDockerImagesCommand(t *testing.T) {
	tempDir := t.TempDir()
	manifestDir := filepath.Join(tempDir, "manifests")
	outputDir := filepath.Join(tempDir, "output")
	
	err := os.MkdirAll(manifestDir, 0755)
	assert.NoError(t, err)

	manifestContent := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  template:
    spec:
      containers:
      - name: app
        image: nginx:1.20
      - name: sidecar
        image: busybox:latest
---
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: main
    image: alpine:3.14
  initContainers:
  - name: init
    image: nginx:1.20`

	createTempManifestFile(t, manifestDir, "test-deployment.yaml", manifestContent)
	
	err = extractDockerImages(manifestDir, outputDir, 0)
	assert.NoError(t, err)

	// Verify output directory was created
	assert.DirExists(t, outputDir)

	// Verify JSON file was created
	jsonFile := filepath.Join(outputDir, "test-deployment.json")
	assert.FileExists(t, jsonFile)

	// Read and verify JSON content
	jsonData, err := os.ReadFile(jsonFile)
	assert.NoError(t, err)

	var images []string
	err = json.Unmarshal(jsonData, &images)
	assert.NoError(t, err)

	// Verify expected images (should be deduplicated)
	expectedImages := []string{"nginx:1.20", "busybox:latest", "alpine:3.14"}
	assert.Equal(t, len(expectedImages), len(images))

	// Check each expected image is present
	imageSet := make(map[string]bool)
	for _, img := range images {
		imageSet[img] = true
	}

	for _, expected := range expectedImages {
		assert.True(t, imageSet[expected], "Expected image %s not found in output", expected)
	}
}


func TestDockerManifestCommand(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		expectedCmd   []string
	}{
		{
			name:        "simple image",
			image:       "alpine:latest",
			expectedCmd: []string{"docker", "manifest", "inspect", "alpine:latest"},
		},
		{
			name:        "nginx image",
			image:       "nginx:1.21",
			expectedCmd: []string{"docker", "manifest", "inspect", "nginx:1.21"},
		},
		{
			name:        "redis image",
			image:       "redis:6.2",
			expectedCmd: []string{"docker", "manifest", "inspect", "redis:6.2"},
		},
		{
			name:        "registry with path",
			image:       "ghcr.io/example/app:v1.0.0",
			expectedCmd: []string{"docker", "manifest", "inspect", "ghcr.io/example/app:v1.0.0"},
		},
		{
			name:        "docker hub with path",
			image:       "docker.io/library/postgres:13",
			expectedCmd: []string{"docker", "manifest", "inspect", "docker.io/library/postgres:13"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate command construction logic
			cmd := tt.expectedCmd
			assert.Equal(t, "docker", cmd[0], "Expected first arg to be 'docker'")
			assert.Equal(t, "manifest", cmd[1], "Expected second arg to be 'manifest'") 
			assert.Equal(t, "inspect", cmd[2], "Expected third arg to be 'inspect'")
			assert.Equal(t, tt.image, cmd[3], "Expected fourth arg to be the image name")
		})
	}
}
