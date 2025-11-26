package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ManifestValidationResult struct {
	ManifestFile string
	Chart        ChartRenderParams
	Error        error
}

type ManifestValidationEngine struct {
	inputChan  chan RenderResult
	resultChan chan ManifestValidationResult
	errorChan  chan ErrorResult

	context         context.Context
	executor        CommandExecutor
	name            string
	schemasDir      string
	workerWaitGroup sync.WaitGroup
}

func (engine *ManifestValidationEngine) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		engine.workerWaitGroup.Add(1)
		go func(workerId int) {
			engine.worker(workerId)
		}(i)
	}
	go engine.allDoneWorker()
}

func (engine *ManifestValidationEngine) allDoneWorker() {
	engine.workerWaitGroup.Wait()
	logEngineDebug(engine.name, -1, "all workers done, closing output channel")
	close(engine.resultChan)
}

func (engine *ManifestValidationEngine) worker(workerId int) {
	defer engine.workerWaitGroup.Done()
	for {
		select {
		case input, ok := <-engine.inputChan:
			if !ok {
				logEngineDebug(engine.name, workerId, "input closed")
				return
			}
			result, err := engine.validateManifest(input.Chart, input.ManifestPath, workerId)
			if err != nil {
				engine.errorChan <- ErrorResult{
					Chart: input.Chart,
					Error: fmt.Errorf("failed to validate manifest %s: %w", input.ManifestPath, err),
				}
				continue
			} else {
				engine.resultChan <- *result
			}

		case <-engine.context.Done():
			logEngineDebug(engine.name, workerId, "context done")
			return
		}
	}
}

func (engine *ManifestValidationEngine) validateManifest(chart ChartRenderParams, manifestFile string, workerId int) (*ManifestValidationResult, error) {

	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		msg := fmt.Sprintf("manifest file does not exist: %s", manifestFile)
		logEngineWarning(engine.name, workerId, msg)
		return nil, fmt.Errorf("manifest file does not exist: %s", manifestFile)
	}
	// Build kubeconform command with absolute schema paths
	schemaPath1 := filepath.Join(engine.schemasDir, "{{.ResourceKind}}_{{.ResourceAPIVersion}}.json")
	schemaPath2 := filepath.Join(engine.schemasDir, "{{.ResourceKind}}-{{.Group}}-{{.ResourceAPIVersion}}.json")

	args := []string{
		"-strict",
		"-summary",
		"-schema-location", "default",
		"-schema-location", "https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json",
		"-schema-location", schemaPath1,
		"-schema-location", schemaPath2,
		"-verbose",
		"-skip", "CustomResourceDefinition",
		"-kubernetes-version", kubernetesVersion,
		"-exit-on-error",
		manifestFile,
	}

	cmd := engine.executor.CommandContext(engine.context,
		"kubeconform", args...,
	)

	// Set working directory to current directory so relative paths work
	if wd, err := os.Getwd(); err == nil {
		cmd.SetDir(wd)
	}

	cmdStr := fmt.Sprintf("%s %s", filepath.Base(cmd.GetPath()), strings.Join(args, " "))
	logEngineDebug(engine.name, workerId, fmt.Sprintf("executing: %s", cmdStr))

	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("kubeconform command failed: %s\nOutput: %s", err.Error(), string(output))
		logEngineWarning(engine.name, workerId, msg)
		return nil, fmt.Errorf("kubeconform command failed: %w", err)
	}

	logEngineDebug(engine.name, workerId, fmt.Sprintf("succeeded: %s\nOutput: %s", cmdStr, string(output)))
	return &ManifestValidationResult{
		ManifestFile: manifestFile,
		Error:        nil,
		Chart:        chart,
	}, nil
}
