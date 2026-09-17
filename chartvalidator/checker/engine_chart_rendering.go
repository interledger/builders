package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ChartRenderingEngine struct {
	inputChan  chan ChartRenderParams
	resultChan chan RenderResult
	errorChan  chan ErrorResult

	outputDir       string
	context         context.Context
	executor        CommandExecutor
	name            string
	retrySleepFn    func(time.Duration)
	workerWaitGroup sync.WaitGroup
	apiVersions     []string
}

type RenderResult struct {
	Chart        ChartRenderParams
	ManifestPath string
}

func (engine *ChartRenderingEngine) Start(workerCount int) {
	if err := recreateOutputDir(engine.outputDir); err != nil {
		msg := fmt.Sprintf("failed to prepare output directory: %s", err.Error())
		logEngineWarning(engine.name, -1, msg)
		panic("This should not happen")
	}

	for i := 0; i < workerCount; i++ {
		engine.workerWaitGroup.Add(1)
		go func(workerId int) {
			engine.worker(workerId)
		}(i)
	}
	go engine.allDoneWorker()
}

func (engine *ChartRenderingEngine) allDoneWorker() {
	engine.workerWaitGroup.Wait()
	logEngineDebug(engine.name, -1, "all workers done, closing output channel")
	close(engine.resultChan)
}

func (engine *ChartRenderingEngine) worker(workerId int) {
	defer engine.workerWaitGroup.Done()

	for {
		select {
		case chart, ok := <-engine.inputChan:
			if !ok {
				logEngineDebug(engine.name, workerId, "input closed")
				return
			}

			result, err := engine.renderSingleChart(chart, workerId)
			if err != nil {
				engine.errorChan <- ErrorResult{Chart: chart, Error: err}
				continue
			}
			engine.resultChan <- *result
		case <-engine.context.Done():
			logEngineDebug(engine.name, workerId, "context done")
			return
		}
	}
}

func (engine *ChartRenderingEngine) renderSingleChart(chart ChartRenderParams, workerId int) (*RenderResult, error) {

	valueFiles := chart.resolvedValueFiles()
	for _, vf := range valueFiles {
		if !engine.executor.FileExists(vf.path) {
			msg := fmt.Sprintf("%s does not exist: %s", vf.label, vf.path)
			logEngineWarning(engine.name, workerId, msg)
			return nil, fmt.Errorf("%s does not exist: %s", vf.label, vf.path)
		}
	}

	// Normalize repoURL so that scheme-less OCI references (which ArgoCD
	// ApplicationSets accept) are treated the same as explicit `oci://` URLs.
	// See normalizeRepoURL for the rationale.
	chart.RepoURL = normalizeRepoURL(chart.RepoURL)

	args := []string{
		"template",
		chart.ChartName,
		resolveChartReference(chart),
	}
	for _, vf := range valueFiles {
		args = append(args, "-f", vf.path)
	}
	args = append(args,
		"--version", chart.ChartVersion,
		"--include-crds",
		"--kube-version", kubernetesVersion,
	)

	if !isOCIRepo(chart.RepoURL) {
		args = append(args, "--repo", chart.RepoURL)
	}

	// Add API versions if any
	for _, apiVer := range engine.apiVersions {
		if apiVer != "" {
			args = append(args, "--api-versions", apiVer)
		}
	}

	logEngineDebug(engine.name, workerId, fmt.Sprintf("helm %s", strings.Join(args, " ")))

	output, err := engine.runHelm(args, workerId)
	if err != nil {
		return nil, fmt.Errorf("helm command failed: %w", err)
	}

	output = sanitizeManifestOutput(output)

	logEngineDebug(engine.name, workerId, fmt.Sprintf("helm %s\t\tCOMPLETED", strings.Join(args, " ")))

	// Create output file path using release name (use absolute path for output)
	absOutputDir, err := filepath.Abs(engine.outputDir)
	if err != nil {
		msg := fmt.Sprintf("failed to get absolute path for output dir: %s", err.Error())
		logEngineWarning(engine.name, workerId, msg)
		return nil, fmt.Errorf("failed to get absolute path for output dir: %w", err)
	}

	randStr := generateRandomString(6)
	filename := fmt.Sprintf("%s_%s.yaml", chart.ChartName, randStr)
	outputPath := filepath.Join(absOutputDir, filename)

	// Write rendered manifests to file
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		msg := fmt.Sprintf("failed to write rendered manifest to file: %s", err.Error())
		logEngineWarning(engine.name, workerId, msg)
		return nil, fmt.Errorf("failed to write rendered manifest to file: %w", err)
	}

	return &RenderResult{Chart: chart, ManifestPath: outputPath}, nil
}

// runHelm executes `helm` and retries when the failure looks like a network
// problem. Every render of a non-OCI chart downloads the repository index
// again, so a run with many charts opens many connections to the same host at
// once. A host such as GitHub Pages answers some of them with a TCP reset,
// which fails the whole pipeline although nothing is wrong with the chart.
func (engine *ChartRenderingEngine) runHelm(args []string, workerId int) ([]byte, error) {
	const maxRetries = 3

	cmdStr := fmt.Sprintf("helm %s", strings.Join(args, " "))

	var output []byte
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			sleepSecs := 1 + rand.Intn(15)
			logEngineWarning(engine.name, workerId, fmt.Sprintf("retrying %s (attempt %d/%d) after %ds", cmdStr, attempt+1, maxRetries+1, sleepSecs))
			engine.sleep(time.Duration(sleepSecs) * time.Second)
			select {
			case <-engine.context.Done():
				return nil, engine.context.Err()
			default:
			}
		}

		cmd := engine.executor.CommandContext(engine.context, "helm", args...)

		// Set working directory to current directory so relative paths work
		if wd, wdErr := os.Getwd(); wdErr == nil {
			cmd.SetDir(wd)
		}

		output, err = cmd.CombinedOutput()
		if err == nil {
			return output, nil
		}

		msg := fmt.Sprintf("helm command failed: %s\nOutput: %s", err.Error(), string(output))
		logEngineWarning(engine.name, workerId, msg)

		if !isTransientHelmError(output) {
			logEngineWarning(engine.name, workerId, fmt.Sprintf("not retrying %s: permanent failure detected", cmdStr))
			break
		}
	}

	return output, err
}

// sleep waits for the given duration. A test replaces retrySleepFn to keep the
// retry path fast; an engine built without one falls back to time.Sleep.
func (engine *ChartRenderingEngine) sleep(d time.Duration) {
	if engine.retrySleepFn != nil {
		engine.retrySleepFn(d)
		return
	}
	time.Sleep(d)
}

// isTransientHelmError returns true if the command output indicates a failure
// that can resolve on retry, such as a reset connection or a 5xx answer from
// the chart repository. The test is the opposite of the one the Docker engine
// makes: most helm template failures are permanent (a bad value, a broken
// template, a missing chart version), so this list names what to retry instead
// of what to give up on.
func isTransientHelmError(output []byte) bool {
	s := strings.ToLower(string(output))
	for _, pattern := range []string{
		"connection reset by peer",
		"connection refused",
		"broken pipe",
		"unexpected eof",
		"i/o timeout",
		"context deadline exceeded",
		"client.timeout",
		"tls handshake",
		"network is unreachable",
		"no route to host",
		"temporary failure in name resolution",
		"server misbehaving",
		"no such host",
		"too many requests",
		"429 ",
		"500 internal server error",
		"502 bad gateway",
		"503 service unavailable",
		"504 gateway",
	} {
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}

func isOCIRepo(repoURL string) bool {
	return strings.HasPrefix(repoURL, "oci://")
}

func resolveChartReference(chart ChartRenderParams) string {
	if isOCIRepo(chart.RepoURL) {
		trimmed := strings.TrimSuffix(chart.RepoURL, "/")
		// An Argo OCI source names the chart twice: repoURL's last segment
		// is the chart, and the `chart` field repeats it. Argo requires
		// both, because it resolves the revision from repoURL alone, and
		// its Application schema requires a `chart` field.
		//
		// Only append ChartName when repoURL is a registry or namespace
		// path that still needs it. Otherwise the reference gets the
		// chart name twice.
		if strings.HasSuffix(trimmed, "/"+chart.ChartName) {
			return trimmed
		}
		return fmt.Sprintf("%s/%s", trimmed, chart.ChartName)
	}
	return chart.ChartName
}

func sanitizeManifestOutput(output []byte) []byte {
	trimmed := bytes.TrimLeft(output, "\n\r\t ")
	manifestMarkers := [][]byte{
		[]byte("---\n"),
		[]byte("apiVersion:"),
		[]byte("# Source:"),
	}

	firstIndex := -1
	for _, marker := range manifestMarkers {
		idx := bytes.Index(trimmed, marker)
		if idx >= 0 && (firstIndex == -1 || idx < firstIndex) {
			firstIndex = idx
		}
	}

	if firstIndex <= 0 {
		return trimmed
	}

	return trimmed[firstIndex:]
}

// Suffix the files just in case two charts end up having the same name
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Int63()%int64(len(charset))]
	}
	return string(b)
}
