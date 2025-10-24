package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DockerImageValidationResult represents the result of validating a single Docker image



type DockerImageValidationEngine struct {
	inputChan  chan ImageExtractionResult
	outputChan chan DockerImageValidationResult

	executor CommandExecutor
	context context.Context

	cache  map[string]DockerImageValidationResult	
	pending map[string]*sync.WaitGroup
	cacheLock sync.RWMutex

	name string

	workerWaitGroup sync.WaitGroup
}

func (engine *DockerImageValidationEngine) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		engine.workerWaitGroup.Add(1)		
		go func(workerId int) {
			engine.worker(workerId)
		}(i)
	}
	go engine.allDoneWorker()
}

func (engine *DockerImageValidationEngine) allDoneWorker() {
	engine.workerWaitGroup.Wait()
	logEngineDebug(engine.name,-1,"all workers done, closing output channel")
	close(engine.outputChan)
}

func (engine *DockerImageValidationEngine) worker(workerId int) {
	defer engine.workerWaitGroup.Done()

	for {
		select {
		case input, ok := <-engine.inputChan:
			if !ok {
				logEngineDebug(engine.name, workerId, "input closed")
				return
			}
			image := input.Image

			// If there is a result pending, then wait for it and return it
			pending_result := engine.waitForPending(input.Chart, image, workerId)
			if pending_result != nil {
				engine.outputChan <- *pending_result
				continue
			}

			// If already cached, return that one
			engine.cacheLock.RLock()
			if result, found := engine.cache[image]; found {
				engine.cacheLock.RUnlock()
				engine.outputChan <- result
				continue
			}
			engine.cacheLock.RUnlock()

			engine.cacheLock.Lock()
			engine.pending[image] = &sync.WaitGroup{}
			pendingWG := engine.pending[image]
			pendingWG.Add(1)			
			engine.cacheLock.Unlock()

			result := engine.validateSingleDockerImage(input.Chart, image, workerId)

			engine.cacheLock.Lock()
				engine.cache[image] = result
				pendingWG.Done()
				delete(engine.pending, image)
			engine.cacheLock.Unlock()
			engine.outputChan <- result

		case <-engine.context.Done():
			logEngineDebug(engine.name,workerId,"context done")
			return
		}
	}
}	

// Should there already be a pending validation for the image, wait for it to complete and return the result
func (engine *DockerImageValidationEngine) waitForPending(chart ChartRenderParams, image string, workerId int) *DockerImageValidationResult {
	engine.cacheLock.RLock()
	if wg, found := engine.pending[image]; found {
		engine.cacheLock.RUnlock()
		logEngineDebug(engine.name, workerId, fmt.Sprintf("waiting for pending: %s", image))
		wg.Wait()
		engine.cacheLock.RLock()
		if result, found := engine.cache[image]; found {
			engine.cacheLock.RUnlock()
			logEngineDebug(engine.name, workerId, fmt.Sprintf("submitting %s result we were waiting for", image))
			return &DockerImageValidationResult{
				Image:  image,
				Exists: result.Exists,
				Error:  result.Error,
				Chart: 	chart,
			}
		}
		logEngineWarning(engine.name, workerId, fmt.Sprintf("even after waiting no result found for %s", image))
		engine.cacheLock.RUnlock()
		return nil
	}
	engine.cacheLock.RUnlock()
	return nil
}

func (engine *DockerImageValidationEngine) validateSingleDockerImage(chart ChartRenderParams, image string, workerId int) DockerImageValidationResult {
	ctx, cancel := context.WithTimeout(engine.context, 2*time.Minute)
	defer cancel()

	args := []string{"manifest", "inspect", image}
	cmd := engine.executor.CommandContext(ctx, "docker", args...)

	// Print the command being executed using interface methods
	cmdStr := fmt.Sprintf("%s %s", filepath.Base(cmd.GetPath()), strings.Join(cmd.GetArgs()[1:], " "))
	logEngineDebug(engine.name, workerId, fmt.Sprintf("executing: %s", cmdStr))

	err := cmd.Run()

	exists := err == nil
	if err != nil {
		logEngineWarning(engine.name, workerId, fmt.Sprintf("failed: %s", cmdStr))
	} else {
		logEngineDebug(engine.name, workerId, fmt.Sprintf("completed: %s", cmdStr))
	}

	return DockerImageValidationResult{
		Image:  image,
		Exists: exists,
		Error:  err,
		Chart: 	chart,
	}

}

// findJSONFiles recursively finds all JSON files in the given directory
func findJSONFiles(dir string) ([]string, error) {
	var jsonFiles []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ".json" {
			jsonFiles = append(jsonFiles, path)
		}

		return nil
	})

	return jsonFiles, err
}

// extractAllImagesFromJSONFiles reads all JSON files and extracts Docker image names
func extractAllImagesFromJSONFiles(jsonFiles []string) ([]string, error) {
	var allImages []string

	for _, jsonFile := range jsonFiles {
		images, err := extractImagesFromJSONFile(jsonFile)
		if err != nil {
			return nil, fmt.Errorf("failed to extract images from %s: %w", jsonFile, err)
		}
		allImages = append(allImages, images...)
	}

	return allImages, nil
}

// extractImagesFromJSONFile reads a single JSON file and extracts the Docker image array
func extractImagesFromJSONFile(jsonFile string) ([]string, error) {
	content, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var images []string
	if err := json.Unmarshal(content, &images); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return images, nil
}

// deduplicateImages removes duplicate images while preserving order
func deduplicateImages(images []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, image := range images {
		if image != "" && !seen[image] {
			seen[image] = true
			unique = append(unique, image)
		}
	}

	// Sort for consistent output
	sort.Strings(unique)
	return unique
}


// createDockerManifestInspectCommand creates the docker command for validating an image
func createDockerManifestInspectCommand(image string) *exec.Cmd {
	return exec.Command("docker", "manifest", "inspect", image)
}
