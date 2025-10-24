package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to create and start a chart rendering engine
func createEngine(mockExecutor *MockCommandExecutor, includeErrorChan bool) *ChartRenderingEngine {
	engine := &ChartRenderingEngine{
		inputChan:  make(chan ChartRenderParams),
		resultChan: make(chan RenderResult),
		outputDir:  "test_output",
		context:    context.Background(),
		executor:   mockExecutor,
	}
	
	if includeErrorChan {
		engine.errorChan = make(chan ErrorResult)
	}
	
	engine.Start(1)
	return engine
}

// Helper function to cleanup engine channels
func cleanupEngine(engine *ChartRenderingEngine) {
	close(engine.inputChan)
	engine.context.Done()
}

func TestRenderBasics(t *testing.T) {
	mockExecutor := createMockExecutor()
	engine := createEngine(mockExecutor, false)
	defer cleanupEngine(engine)

	testChart := createTestChart()
	engine.inputChan <- testChart

	result := <-engine.resultChan
	assertChartFieldsMatch(t, testChart, result.Chart)

	// Verify the command that was executed
	expectedCommand := "helm template test-chart --release-name test-chart --repo https://example.com/charts -f values.yaml -f override.yaml --version 1.0.0 --include-crds"
	actualCommand := mockExecutor.GetFullCommand()
	assert.Equal(t, expectedCommand, actualCommand)
}

func TestRenderBaseFileNotExist(t *testing.T) {
	mockExecutor := createMockExecutor()
	mockExecutor.FileExistsMap = map[string]bool{
		"values.yaml":   false,
		"override.yaml": true,
	}

	engine := createEngine(mockExecutor, true)
	defer cleanupEngine(engine)

	testChart := createTestChart()
	engine.inputChan <- testChart

	errorResult := <-engine.errorChan
	assert.Equal(t, errorResult.Chart.ChartName, testChart.ChartName)
	assert.NotNil(t, errorResult.Error)
	assert.Contains(t, errorResult.Error.Error(), "base values file does not exist")
}