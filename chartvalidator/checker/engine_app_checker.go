package main

import (
	"context"
	"fmt"
	"sync"
)

type AppCheckInstruction struct {
	Chart ChartRenderParams
}

type AppCheckResult struct {
	Chart ChartRenderParams
	Image string
	Error error
}

type AppCheckerEngine struct {
	inputChan  chan AppCheckInstruction
	resultChan chan AppCheckResult
	errorChan  chan ErrorResult

	ChartRenderingEngine  *ChartRenderingEngine
	ManifestValidationEngine *ManifestValidationEngine
	ImageExtractionEngine   *ImageExtractionEngine
	DockerValidationEngine   *DockerImageValidationEngine

	context    context.Context
	executor   CommandExecutor

	workerWaitGroup sync.WaitGroup

	name string
}

func NewAppCheckerEngine(context context.Context, outputDir string) *AppCheckerEngine {

	errorChan := make(chan ErrorResult)

	cre := ChartRenderingEngine{
		inputChan: make(chan ChartRenderParams),
		resultChan: make(chan RenderResult),
		errorChan: errorChan,
		outputDir: outputDir,
		context: context,
		executor: &RealCommandExecutor{},
		name: "ChartRenderer",
	}

	mve := ManifestValidationEngine{
		inputChan: cre.resultChan,
		resultChan: make(chan ManifestValidationResult),
		errorChan: errorChan,
		context: context,
		executor: &RealCommandExecutor{},
		name: "ManifestValidator",
		workerWaitGroup: sync.WaitGroup{},
	}

	iee := ImageExtractionEngine{
		inputChan: mve.resultChan,
		outputChan: make(chan ImageExtractionResult),
		errorChan: errorChan,
		context: context,
		name: "ImageExtractor",
		workerWaitGroup: sync.WaitGroup{},
	}

	dve := DockerImageValidationEngine{
		inputChan: iee.outputChan,
		outputChan: make(chan DockerImageValidationResult),
		context: context,
		executor: &RealCommandExecutor{},
		name: "DockerValidator",
		cache: map[string]DockerImageValidationResult{},
		pending: map[string]*sync.WaitGroup{},
		cacheLock: sync.RWMutex{},
		workerWaitGroup: sync.WaitGroup{},
	}
	
	return &AppCheckerEngine{
		inputChan:  make(chan AppCheckInstruction),
		resultChan: make(chan AppCheckResult),
		errorChan:  make(chan ErrorResult),

		context:    context,
		executor:   &RealCommandExecutor{},

		ChartRenderingEngine: &cre,
		ManifestValidationEngine: &mve,
		ImageExtractionEngine:   &iee,
		DockerValidationEngine:   &dve,

		name: "AppChecker",
	}
}

func (engine *AppCheckerEngine) allDoneWorker() {
	engine.workerWaitGroup.Wait()
	logEngineDebug(engine.name,-1,"all workers done, closing output channel")	
	close(engine.resultChan)
}

func (engine *AppCheckerEngine) Start(workerCount int) {

	// Fire up the engines
	engine.ChartRenderingEngine.Start(workerCount)
	engine.ManifestValidationEngine.Start(workerCount)
	engine.ImageExtractionEngine.Start(workerCount)
	engine.DockerValidationEngine.Start(workerCount)

	// Pour the input instructions into the chart renderer
	engine.workerWaitGroup.Add(1)
	go engine.pumpAppCheckInstructionsToChartRenderer()
	engine.workerWaitGroup.Add(1)	
	go engine.pumpOutputsToAppCheckResults()

	go engine.allDoneWorker()
}

func (engine *AppCheckerEngine) pumpOutputsToAppCheckResults() {
	defer engine.workerWaitGroup.Done()
	for dockerResult := range engine.DockerValidationEngine.outputChan {
		if dockerResult.Error != nil {
			engine.resultChan <- AppCheckResult{
				Chart: dockerResult.Chart,
				Image: dockerResult.Image,
				Error: dockerResult.Error,
			}
			continue
		} else {
			var err error = nil
			if !dockerResult.Exists {
				err = fmt.Errorf("docker image does not exist: %s", dockerResult.Image)
			}
			engine.resultChan <- AppCheckResult{
				Chart: dockerResult.Chart,
				Image: dockerResult.Image,
				Error: err,
			}
		}
	}
	logEngineDebug(engine.name, -1, "docker validation output closed")
}

func (engine *AppCheckerEngine) pumpAppCheckInstructionsToChartRenderer() {
	defer engine.workerWaitGroup.Done()
	for instruction := range engine.inputChan {
		engine.ChartRenderingEngine.inputChan <- ChartRenderParams{
			Env: instruction.Chart.Env,
			ChartName: instruction.Chart.ChartName,
			RepoURL: instruction.Chart.RepoURL,
			ChartVersion: instruction.Chart.ChartVersion,
			BaseValuesFile: instruction.Chart.BaseValuesFile,
			ValuesOverride: instruction.Chart.ValuesOverride,
		}
	}
	close(engine.ChartRenderingEngine.inputChan)
}