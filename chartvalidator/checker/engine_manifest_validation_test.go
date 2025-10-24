package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestValidationEngine(t *testing.T) {
	mockExecutor := createManifestValidationMockExecutor()
	engine := createManifestValidationEngine(mockExecutor)
	engine.Start(1)

	testManifestFile := "test_data/example.yaml"
	sendRenderResultToEngine(engine, testManifestFile)

	result := <-engine.resultChan

	// Verify no error occurred
	assert.NoError(t, result.Error, "Expected no error during manifest validation")

	// Verify manifest file path is correct
	assert.Equal(t, testManifestFile, result.ManifestFile, "Expected correct manifest file path")

	// Verify the command that was executed
	expectedCommand := "kubeconform -strict -summary -schema-location default -schema-location https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json -schema-location ci/schemas/{{ .ResourceKind }}_{{ .ResourceAPIVersion }}.json -verbose -exit-on-error test_data/example.yaml"
	assertCommandExecution(t, mockExecutor, expectedCommand)

	close(engine.inputChan)
}

func TestManifestValidationEngineMultipleFiles(t *testing.T) {
	verboseLogging = true

	testCases := []struct {
		name         string
		manifestPath string
	}{
		{
			name:         "deployment manifest",
			manifestPath: "test_data/deployment.yaml",
		},
		{
			name:         "service manifest",
			manifestPath: "test_data/service.yaml",
		},
		{
			name:         "configmap manifest",
			manifestPath: "test_data/configmap.yaml",
		},
		{
			name:         "deployment manifest2",
			manifestPath: "test_data/deployment.yaml",
		},
		{
			name:         "service manifest2",
			manifestPath: "test_data/service.yaml",
		},
		{
			name:         "configmap manifest2",
			manifestPath: "test_data/configmap.yaml",
		},		
	}

	mockExecutor := createManifestValidationMockExecutor()
	engine := createManifestValidationEngine(mockExecutor)
	engine.Start(2)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			sendRenderResultToEngine(engine, tc.manifestPath)

			var result ManifestValidationResult
			select {
			case result = <-engine.resultChan:
				t.Log("ok")
			case errResult := <-engine.errorChan:
				t.Fatalf("Expected no error for manifest %s, got error: %v", tc.manifestPath, errResult.Error)
			}

			// Verify no error occurred
			assert.NoError(t, result.Error, "Expected no error during manifest validation")

			// Verify manifest file path is correct
			assert.Equal(t, tc.manifestPath, result.ManifestFile, "Expected correct manifest file path")

			// Verify command contains the manifest path
			actualCommand := mockExecutor.GetFullCommand()
			assert.Contains(t, actualCommand, tc.manifestPath, "Expected command to contain manifest path")

		})
	}
	close(engine.inputChan)
	engine.workerWaitGroup.Wait()
}

func TestManifestValidationEngineWithError(t *testing.T) {
	// Create mock executor that returns an error
	mockExecutor := createMockExecutorWithBehavior(func() error {
		return assert.AnError
	})
	mockExecutor.Output = []byte("validation failed")

	engine := createManifestValidationEngine(mockExecutor)
	engine.Start(1)

	testManifestFile := "test_data/invalid.yaml"
	sendRenderResultToEngine(engine, testManifestFile)

	// Should receive an error result
	select {
	case result := <-engine.resultChan:
		// If we get a result, it should have an error
		assert.Error(t, result.Error, "Expected an error for invalid manifest")
		assert.Equal(t, testManifestFile, result.ManifestFile, "Expected correct manifest file path even with error")
	case errorResult := <-engine.errorChan:
		// Or we might get an error result
		assert.Error(t, errorResult.Error, "Expected an error for invalid manifest")
	}

	close(engine.inputChan)
	engine.workerWaitGroup.Wait()
}