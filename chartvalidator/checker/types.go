package main

import (
	"os/exec"
	"sync"
)

type ErrorResult struct {
	Chart ChartRenderParams
	Error error
}

type DockerImageValidationResult struct {
	Chart  ChartRenderParams
	Image  string
	Exists bool
	Error  error
}

type ImageExtractionResult struct {
	Chart        ChartRenderParams
	ManifestFile string
	Image        string
}

// ChartRenderParams represents a Helm chart configuration extracted from ApplicationSet files
type ChartRenderParams struct {
	Env            string `json:"env"`
	ChartName      string `json:"chartName"`
	RepoURL        string `json:"repoURL"`
	ChartVersion   string `json:"chartVersion"`
	BaseValuesFile string `json:"baseValuesFile"`
	ValuesOverride string `json:"valuesOverride"`
	// ValueFiles is an ordered list of values files, used by charts discovered
	// in Applications: their helm.valueFiles can hold any number of entries,
	// whereas an ApplicationSet list element always carries the fixed
	// BaseValuesFile/ValuesOverride pair above. When set it replaces that pair.
	ValueFiles []string `json:"valueFiles,omitempty"`
}

// valueFileRef is a values file plus the wording used to report it missing.
type valueFileRef struct {
	path  string
	label string
}

// resolvedValueFiles returns the values files to pass to helm, in order.
func (c ChartRenderParams) resolvedValueFiles() []valueFileRef {
	if len(c.ValueFiles) > 0 {
		refs := make([]valueFileRef, 0, len(c.ValueFiles))
		for _, p := range c.ValueFiles {
			refs = append(refs, valueFileRef{path: p, label: "values file"})
		}
		return refs
	}
	return []valueFileRef{
		{path: c.BaseValuesFile, label: "base values file"},
		{path: c.ValuesOverride, label: "values override file"},
	}
}

// task represents a validation task with a chart and command
type task struct {
	Chart ChartRenderParams
	Cmd   *exec.Cmd
}

// imageCheck represents the result of checking if a Docker image exists
type imageCheck struct {
	Chart   ChartRenderParams
	Image   string
	Present bool
	Error   error
}

// validationResult represents the result of a kubeconform validation
type validationResult struct {
	Chart ChartRenderParams
	RC    int
	Out   string
	Err   string
}

// validationFailure represents a failed validation with chart and details
type validationFailure struct {
	Chart  ChartRenderParams
	RC     int
	Output string
}

// imageCheckSetup manages image checking infrastructure
type imageCheckSetup struct {
	inputPipe  chan *imageCheck
	resultPipe chan *imageCheck
	results    map[string]*imageCheck
	workerWg   sync.WaitGroup
	resultsWg  sync.WaitGroup
}
