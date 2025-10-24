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
	Chart       ChartRenderParams
	ManifestFile string
	Image       string
}

// ChartRenderParams represents a Helm chart configuration extracted from ApplicationSet files
type ChartRenderParams struct {
	Env            string `json:"env"`
	ChartName      string `json:"chartName"`
	RepoURL        string `json:"repoURL"`
	ChartVersion   string `json:"chartVersion"`
	BaseValuesFile string `json:"baseValuesFile"`
	ValuesOverride string `json:"valuesOverride"`
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
	inputPipe   chan *imageCheck
	resultPipe  chan *imageCheck
	results     map[string]*imageCheck
	workerWg    sync.WaitGroup
	resultsWg   sync.WaitGroup
}