package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Consumes manifest files from inputChan, extracts Docker images, and sends results to outputChan
type ImageExtractionEngine struct {
	// Each string should be a path to a manifest file
	inputChan chan ManifestValidationResult
	outputChan chan ImageExtractionResult
	errorChan  chan ErrorResult

	context context.Context
	workerWaitGroup sync.WaitGroup
	name string
}

func (engine *ImageExtractionEngine) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		engine.workerWaitGroup.Add(1)		
		go func(workerId int) {
			engine.worker(workerId)
		}(i)
	}
	go engine.allDoneWorker()
}

func (engine *ImageExtractionEngine) allDoneWorker() {
	logEngineDebug(engine.name,-1, "waiting for workers to finish")
	engine.workerWaitGroup.Wait()
	logEngineDebug(engine.name,-1,"all workers done, closing output channel")	
	close(engine.outputChan)
}

func (engine *ImageExtractionEngine) worker(workerId int) {
	defer engine.workerWaitGroup.Done()
	for {
		select {
		case input, ok := <-engine.inputChan:
			if !ok {
				logEngineDebug(engine.name, workerId, "input closed")
				return
			}
			images, err := engine.extractImagesFromFile(input.ManifestFile, workerId)
			if err != nil {
				logEngineWarning(engine.name, workerId, fmt.Sprintf("failed to extract images from %s: %v", input.ManifestFile, err))
				engine.errorChan <- ErrorResult{
					Chart: input.Chart,
					Error:  fmt.Errorf("failed to extract images from %s: %w", input.ManifestFile, err),
				}
				continue
			} else {
				uniqueImages := removeDuplicates(images)
				// Send each extracted image as a separate result for the next step
				logEngineDebug(engine.name, workerId, fmt.Sprintf("extracted %d images from %s", len(uniqueImages), input.ManifestFile))
				for _, img := range uniqueImages {
					engine.outputChan <- ImageExtractionResult{
						Chart: input.Chart,
						ManifestFile: input.ManifestFile,
						Image:       img,
					}
				}
			}
		case <-engine.context.Done():
			logEngineDebug(engine.name, workerId, "context done")
			return
		}
	}
}

func (engine *ImageExtractionEngine) extractImagesFromFile(file string, workerId int) ([]string, error) {
	// Read the manifest file
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Split content into multiple YAML documents (in case of multi-document files)
	documents := strings.Split(string(content), "\n---\n")
	var allImages []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		// Extract images from this document
		images, err := extractImageFromManifest(doc, workerId)
		if err != nil {
			// Don't fail the entire file for one bad document, just log and continue
			logEngineWarning(engine.name, workerId, fmt.Sprintf("failed to extract images from document in %s: %v", file, err))
			continue
		}

		allImages = append(allImages, images...)
	}

	return allImages, nil
}


// extractDockerImages extracts Docker images from all manifest files in the specified directory
// and saves the results as JSON files in the output directory
func extractDockerImages(manifestDir, outputDir string, workerId int) error {
	// Check if the source directory exists
	if _, err := os.Stat(manifestDir); os.IsNotExist(err) {
		return fmt.Errorf("directory %s does not exist", manifestDir)
	}

	// Remove and recreate output directory
	if err := recreateOutputDir(outputDir); err != nil {
		return fmt.Errorf("failed to prepare output directory: %w", err)
	}

	// Find all YAML files in the directory
	yamlFiles, err := findYAMLFiles(manifestDir)
	if err != nil {
		return fmt.Errorf("failed to find YAML files in %s: %w", manifestDir, err)
	}

	if len(yamlFiles) == 0 {
		logEngineWarning("ImageExtractor", -1, fmt.Sprintf("No YAML files found in %s", manifestDir))
		return nil
	}

	logEngineDebug("ImageExtractor", -1, fmt.Sprintf("Extracting Docker images from %d YAML files in %s", len(yamlFiles), manifestDir))

	for _, yamlFile := range yamlFiles {
		if err := extractImagesFromFile(yamlFile, manifestDir, outputDir, workerId); err != nil {
			logEngineWarning("ImageExtractor", -1, fmt.Sprintf("failed to extract images from %s: %v", yamlFile, err))
			continue
		}
	}

	logEngineDebug("ImageExtractor", -1, fmt.Sprintf("Docker image extraction complete. JSON files written to %s/", outputDir))
	return nil
}

// extractImagesFromFile extracts Docker images from a single manifest file and saves to JSON
func extractImagesFromFile(yamlFile, manifestDir, outputDir string, workerId int) error {
	// Read the manifest file
	content, err := os.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Split content into multiple YAML documents (in case of multi-document files)
	documents := strings.Split(string(content), "\n---\n")
	var allImages []string

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		// Extract images from this document
		images, err := extractImageFromManifest(doc, workerId)
		if err != nil {
			// Don't fail the entire file for one bad document, just log and continue
			logEngineWarning("ImageExtractor", workerId, fmt.Sprintf("failed to extract images from document in %s: %v", yamlFile, err))
			continue
		}

		allImages = append(allImages, images...)
	}

	// Remove duplicates from the image list
	uniqueImages := removeDuplicates(allImages)

	// Create output file name based on manifest file name
	relPath, err := filepath.Rel(manifestDir, yamlFile)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Replace file extension with .json and replace path separators with underscores
	jsonFileName := strings.ReplaceAll(relPath, string(filepath.Separator), "_")
	jsonFileName = strings.TrimSuffix(jsonFileName, filepath.Ext(jsonFileName)) + ".json"
	outputPath := filepath.Join(outputDir, jsonFileName)

	// Create JSON output
	jsonData, err := json.MarshalIndent(uniqueImages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write JSON file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file %s: %w", outputPath, err)
	}

	logEngineDebug("ImageExtractor", -1, fmt.Sprintf("Extracted %d unique images from %s -> %s", len(uniqueImages), relPath, jsonFileName))
	return nil
}


func extractImagesFromDeployment(manifest map[string]interface{}) ([]string, error) {
	// Validate this is a Deployment
	kind, ok := manifest["kind"].(string)
	if !ok || kind != "Deployment" {
		return nil, fmt.Errorf("not a Deployment manifest")
	}

	// Extract the pod section and use extractImagesFromPod to do the work
	spec, ok := manifest["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing spec in Deployment")
	}
	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing template in Deployment spec")
	}
	_, ok = template["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing pod spec in Deployment template")
	}

	return extractImagesFromPod(template)
}

func extractImagesFromDaemonSet(manifest map[string]interface{}) ([]string, error) {
	// Validate this is a DaemonSet
	kind, ok := manifest["kind"].(string)
	if !ok || kind != "DaemonSet" {
		return nil, fmt.Errorf("not a DaemonSet manifest")
	}

	// Extract the pod section and use extractImagesFromPod to do the work
	spec, ok := manifest["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing spec in DaemonSet")
	}
	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing template in DaemonSet spec")
	}
	_, ok = template["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing pod spec in DaemonSet template")
	}

	return extractImagesFromPod(template)
}

func extractImagesFromStatefulSet(manifest map[string]interface{}) ([]string, error) {
	// Validate this is a StatefulSet
	kind, ok := manifest["kind"].(string)
	if !ok || kind != "StatefulSet" {
		return nil, fmt.Errorf("not a StatefulSet manifest")
	}

	// Extract the pod section and use extractImagesFromPod to do the work
	spec, ok := manifest["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing spec in StatefulSet")
	}
	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing template in StatefulSet spec")
	}
	_, ok = template["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing pod spec in StatefulSet template")
	}

	return extractImagesFromPod(template)
}

func extractImagesFromPod(manifest map[string]interface{}) ([]string, error) {
	images := []string{}

	spec, ok := manifest["spec"].(map[string]interface{})
	if !ok {
		return images, nil // No spec found
	}

	// Check containers
	if containers, ok := spec["containers"].([]interface{}); ok {
		for _, c := range containers {
			if cMap, ok := c.(map[string]interface{}); ok {
				if img, ok := cMap["image"].(string); ok {
					images = append(images, img)
				}
			}
		}
	}

	// Check initContainers
	if initContainers, ok := spec["initContainers"].([]interface{}); ok {
		for _, c := range initContainers {
			if cMap, ok := c.(map[string]interface{}); ok {
				if img, ok := cMap["image"].(string); ok {
					images = append(images, img)
				}
			}
		}
	}

	return images, nil
}


// Extracts all of the docker images references from a given Kubernetes manifest.
// This function makes the assumption that only a single manifest is provided at
// a time, and that it is a Pod or Pod-like object (e.g. Deployment, DaemonSet).
func extractImageFromManifest(manifest string, workerId int) ([]string, error) {
	imagesFound := []string{}

	// Parse the YAML manifest into a generic map.
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(manifest), &doc); err != nil {
		return imagesFound, fmt.Errorf("failed to parse YAML: %w", err)
	}

	kind, ok := doc["kind"].(string)
	if !ok {
		return imagesFound, fmt.Errorf("manifest missing 'kind' field")
	}

	logEngineDebug("ImageExtractor", workerId, fmt.Sprintf("Inspecting %s %s", kind, fmt.Sprint(doc["metadata"].(map[string]interface{})["name"])))

	switch kind {
	case "Pod":

		images, err := extractImagesFromPod(doc)
		if err != nil {
			return imagesFound, err
		}
		imagesFound = append(imagesFound, images...)
	case "Deployment":
		images, err := extractImagesFromDeployment(doc)
		if err != nil {
			return imagesFound, err
		}
		imagesFound = append(imagesFound, images...)
	case "DaemonSet":
		images, err := extractImagesFromDaemonSet(doc)
		if err != nil {
			return imagesFound, err
		}
		imagesFound = append(imagesFound, images...)	

	case "StatefulSet":
		images, err := extractImagesFromStatefulSet(doc)
		if err != nil {
			return imagesFound, err
		}
		imagesFound = append(imagesFound, images...)

	default:
		// For other kinds, we currently do not extract images.
		logEngineDebug("ImageExtractor", workerId, fmt.Sprintf("Skipping image extraction for %s %s", kind, fmt.Sprint(doc["metadata"].(map[string]interface{})["name"])))
		return imagesFound, nil
	}

	return imagesFound, nil
	
}
