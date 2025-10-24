package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
)


type ChartRenderingEngine struct {
	inputChan  chan ChartRenderParams
	resultChan chan RenderResult
	errorChan  chan ErrorResult

	outputDir  string
	context    context.Context
	executor   CommandExecutor
	name	   string
	workerWaitGroup sync.WaitGroup
}

type RenderResult struct {
	Chart            ChartRenderParams
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
	logEngineDebug(engine.name,-1,"all workers done, closing output channel")	
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

	if !engine.executor.FileExists(chart.BaseValuesFile) {
		msg := fmt.Sprintf("base values file does not exist: %s", chart.BaseValuesFile)
		logEngineWarning(engine.name, workerId, msg)
		return nil, fmt.Errorf("base values file does not exist: %s", chart.BaseValuesFile)
	}
	if !engine.executor.FileExists(chart.ValuesOverride) {
		msg := fmt.Sprintf("values override file does not exist: %s", chart.ValuesOverride)
		logEngineWarning(engine.name, workerId, msg)
		return nil, fmt.Errorf("values override file does not exist: %s", chart.ValuesOverride)
	}

	args := []string{
		"template", chart.ChartName,
		"--release-name", chart.ChartName,
		"--repo", chart.RepoURL,
		"-f", chart.BaseValuesFile,
		"-f", chart.ValuesOverride,
		"--version", chart.ChartVersion,
		"--include-crds",
	}

	logEngineDebug(engine.name, workerId, fmt.Sprintf("helm %s", strings.Join(args, " ")))
	cmd := engine.executor.CommandContext(engine.context, "helm", args...)
	
	// Set working directory to current directory so relative paths work
	if wd, err := os.Getwd(); err == nil {
		cmd.SetDir(wd)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("helm command failed: %s\nOutput: %s", err.Error(), string(output))
		logEngineWarning(engine.name, workerId, msg)
		return nil, fmt.Errorf("helm command failed: %w", err)
	}

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

// Suffix the files just in case two charts end up having the same name
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Int63()%int64(len(charset))]
	}
	return string(b)
}
