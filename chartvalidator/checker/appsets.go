package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// findChartsInAppsets scans ApplicationSet files and extracts chart information
func findChartsInAppsets(envDir, selectedEnv string) ([]ChartRenderParams, error) {
	const suffix = "appset.yaml"
	var out []ChartRenderParams

	fmt.Println("Scanning environments in", envDir)

	if selectedEnv != "" {
		envPath := filepath.Join(envDir, selectedEnv)
		ok, err := existsDir(envPath)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("environment %q not found in %s", selectedEnv, envDir)
		}
		ch, err := processEnvironment(selectedEnv, envPath, suffix)
		if err != nil {
			return nil, err
		}
		return ch, nil
	}

	entries, err := os.ReadDir(envDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		envName := e.Name()
		envPath := filepath.Join(envDir, envName)
		ch, err := processEnvironment(envName, envPath, suffix)
		if err != nil {
			return nil, err
		}
		out = append(out, ch...)
	}
	return out, nil
}

// processEnvironment extracts charts from a single environment directory
func processEnvironment(envName, envPath, suffix string) ([]ChartRenderParams, error) {
	appsetsPath := filepath.Join(envPath, "appsets")
	ok, err := existsDir(appsetsPath)
	if err != nil || !ok {
		return []ChartRenderParams{}, err
	}

	files, err := listAppsetFiles(appsetsPath, suffix)
	if err != nil {
		return nil, err
	}

	var charts []ChartRenderParams
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var node any
		if err := yaml.Unmarshal(data, &node); err != nil {
			return nil, fmt.Errorf("failed to parse YAML %s: %w", f, err)
		}
		elems := extractElements(node)
		for _, el := range elems {
			charts = append(charts, extractChartInfo(el, envName))
		}
	}
	return charts, nil
}

// listAppsetFiles returns all files ending with the given suffix in the directory
func listAppsetFiles(dir, suffix string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, suffix) {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out, nil
}

// existsDir checks if a directory exists
func existsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// extractElements extracts the list elements from an ApplicationSet document
func extractElements(doc any) []map[string]any {
	// Navigate: spec.generators[0].list.elements
	m, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	spec, _ := m["spec"].(map[string]any)
	if spec == nil {
		return nil
	}
	gens, _ := spec["generators"].([]any)
	if len(gens) == 0 {
		return nil
	}
	gen0, _ := gens[0].(map[string]any)
	if gen0 == nil {
		return nil
	}
	lst, _ := gen0["list"].(map[string]any)
	if lst == nil {
		return nil
	}
	elems, _ := lst["elements"].([]any)
	if len(elems) == 0 {
		return nil
	}
	var out []map[string]any
	for _, e := range elems {
		if mm, ok := e.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

// extractChartInfo extracts Chart information from an ApplicationSet element
func extractChartInfo(el map[string]any, env string) ChartRenderParams {
	return ChartRenderParams{
		Env:            env,
		ChartName:      str(el["chartName"]),
		RepoURL:        str(el["repoURL"]),
		ChartVersion:   str(el["chartVersion"]),
		BaseValuesFile: srcPrefix + str(el["baseValuesFile"]),
		ValuesOverride: srcPrefix + str(el["valuesOverride"]),
	}
}

// str converts any value to string, handling nil safely
func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}