package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to create and start a chart rendering engine
func createEngine(mockExecutor *MockCommandExecutor, includeErrorChan bool) *ChartRenderingEngine {
	engine := &ChartRenderingEngine{
		inputChan:   make(chan ChartRenderParams),
		resultChan:  make(chan RenderResult),
		outputDir:   "test_output",
		context:     context.Background(),
		executor:    mockExecutor,
		apiVersions: []string{"something", "something-else"},
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
	expectedCommand := "helm template test-chart test-chart -f values.yaml -f override.yaml --version 1.0.0 --include-crds --kube-version 1.33.0 --repo https://example.com/charts --api-versions something --api-versions something-else"
	actualCommand := mockExecutor.GetFullCommand()
	assert.Equal(t, expectedCommand, actualCommand)
}

func TestRenderOCIRepo(t *testing.T) {
	mockExecutor := createMockExecutor()
	mockExecutor.Output = []byte("Pulled: europe-west4-docker.pkg.dev/wallet-dev-462809/interledger-helm-charts/test-chart:1.0.0\nDigest: sha256:abc123\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")
	engine := createEngine(mockExecutor, false)
	defer cleanupEngine(engine)

	testChart := createTestChart()
	testChart.RepoURL = "oci://europe-west4-docker.pkg.dev/wallet-dev-462809/interledger-helm-charts"
	engine.inputChan <- testChart

	result := <-engine.resultChan
	assertChartFieldsMatch(t, testChart, result.Chart)

	// OCI charts should be templated using the full OCI chart reference, not --repo.
	expectedCommand := "helm template test-chart oci://europe-west4-docker.pkg.dev/wallet-dev-462809/interledger-helm-charts/test-chart -f values.yaml -f override.yaml --version 1.0.0 --include-crds --kube-version 1.33.0 --api-versions something --api-versions something-else"
	actualCommand := mockExecutor.GetFullCommand()
	assert.Equal(t, expectedCommand, actualCommand)

	renderedManifest, err := os.ReadFile(result.ManifestPath)
	assert.NoError(t, err)
	assert.Equal(t, "---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n", string(renderedManifest))
}

// Scheme-less repoURLs (as used in ArgoCD ApplicationSets that point at GAR /
// other OCI registries without the oci:// prefix) must be templated as OCI,
// matching the behaviour of explicit `oci://` URLs.
func TestRenderOCIRepoNoScheme(t *testing.T) {
	mockExecutor := createMockExecutor()
	mockExecutor.Output = []byte("---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")
	engine := createEngine(mockExecutor, false)
	defer cleanupEngine(engine)

	testChart := createTestChart()
	testChart.RepoURL = "europe-west4-docker.pkg.dev/wallet-dev-462809/interledger-helm-charts"
	engine.inputChan <- testChart

	result := <-engine.resultChan
	// Engine normalises RepoURL, so compare against the normalised form.
	assert.Equal(t, "oci://europe-west4-docker.pkg.dev/wallet-dev-462809/interledger-helm-charts", result.Chart.RepoURL)

	expectedCommand := "helm template test-chart oci://europe-west4-docker.pkg.dev/wallet-dev-462809/interledger-helm-charts/test-chart -f values.yaml -f override.yaml --version 1.0.0 --include-crds --kube-version 1.33.0 --api-versions something --api-versions something-else"
	actualCommand := mockExecutor.GetFullCommand()
	assert.Equal(t, expectedCommand, actualCommand)
}

// repoURL can already carry the chart name as its last segment. Argo's OCI
// sources work this way, and they set the `chart` field to that same name.
// The chart reference must not repeat the chart name in that case.
func TestRenderOCIRepoChartEmbeddedInRepoURL(t *testing.T) {
	mockExecutor := createMockExecutor()
	mockExecutor.Output = []byte("Pulled: ghcr.io/interledger/charts/merchant:0.0.4\nDigest: sha256:abc123\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")
	engine := createEngine(mockExecutor, false)
	defer cleanupEngine(engine)

	testChart := createTestChart()
	testChart.ChartName = "merchant"
	testChart.RepoURL = "oci://ghcr.io/interledger/charts/merchant"
	testChart.ChartVersion = "0.0.4"
	engine.inputChan <- testChart

	result := <-engine.resultChan
	assertChartFieldsMatch(t, testChart, result.Chart)

	expectedCommand := "helm template merchant oci://ghcr.io/interledger/charts/merchant -f values.yaml -f override.yaml --version 0.0.4 --include-crds --kube-version 1.33.0 --api-versions something --api-versions something-else"
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
