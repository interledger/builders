package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppCheckerEnginePropagatesRenderErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outputDir := t.TempDir()

	engine := NewAppCheckerEngine(ctx, outputDir, nil)

	mockExecutor := createMockExecutor()
	mockExecutor.Error = fmt.Errorf("helm exploded")
	mockExecutor.Output = []byte("nope")
	mockExecutor.FileExistsMap = map[string]bool{
		"values.yaml":   true,
		"override.yaml": true,
	}

	engine.ChartRenderingEngine.executor = mockExecutor

	engine.Start(1)

	testChart := ChartRenderParams{
		Env:            "test",
		ChartName:      "chart",
		RepoURL:        "https://example.com/charts",
		ChartVersion:   "1.0.0",
		BaseValuesFile: "values.yaml",
		ValuesOverride: "override.yaml",
	}

	go func() {
		engine.inputChan <- AppCheckInstruction{Chart: testChart}
		close(engine.inputChan)
	}()

	var (
		result AppCheckResult
		ok     bool
	)

	select {
	case result, ok = <-engine.resultChan:
		if !ok {
			t.Fatalf("result channel closed before emitting result")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for app checker result (possible deadlock)")
	}

	assert.Equal(t, testChart.ChartName, result.Chart.ChartName)
	assert.Error(t, result.Error)
	if result.Error != nil {
		assert.Contains(t, result.Error.Error(), "helm command failed")
	}

	select {
	case _, ok = <-engine.resultChan:
		if ok {
			t.Fatalf("expected result channel to close after delivering error result")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for result channel to close")
	}
}
