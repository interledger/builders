package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
)

var srcPrefix string = "../"
var verboseLogging bool = false

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "run-checks":
		runChartChecksCommand(args)
	case "render-only":
		runRenderOnlyCommand(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: chart-checker <command> [flags]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  run-checks    Runs all available checks on the charts for given environment.")
	fmt.Println("  render-only   Renders the charts for the given environment without performing validations.")
	fmt.Println("  help          Displays this help message.")
	fmt.Println("")
	fmt.Println("Use 'run-manifest-checks <command> -h' to see command-specific flags.")
}



func runChartChecksCommand(args []string) {
	fs := flag.NewFlagSet("run-checks", flag.ExitOnError)

	var (
		singleEnv = fs.String("env", "", "Only process this environment (folder name under -envdir).")
		envDir    = fs.String("envdir", "../env", "Base directory containing environment folders.")
		outputDir = fs.String("output", "manifests", "Output directory for rendered charts.")
		verbose   = fs.Bool("v", false, "Enable verbose logging.")
	)	

	fs.Usage = func() {
		fmt.Println("Usage: run-manifest-checks run-checks [flags]")
		fmt.Println("")
		fmt.Println("Will run a series of checks against all charts found in the ApplicationSets in the specified environment.")
		fmt.Println("Steps are as follows:")
		fmt.Println(" 1. Find all charts referenced in ApplicationSets in the specified environment.")
		fmt.Println(" 2. Render each chart with its values using Helm.")
		fmt.Println(" 3. Validate the rendered manifests using kubeconform.")
		fmt.Println(" 4. Extract Docker image references from the manifests.")
		fmt.Println(" 5. Validate that each Docker image exists in the registry.")
		fmt.Println("")
		fmt.Println("Docker needs to be authenticated to the registries used by the charts for image validation to work.")
		fmt.Println("")		
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	verboseLogging = *verbose

	if err := runAllChartChecks(*singleEnv, *envDir, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error running chart checks: %v\n", err)
		os.Exit(1)
	}

}

func runRenderOnlyCommand(args []string) {
	fs := flag.NewFlagSet("render-only", flag.ExitOnError)

	var (
		singleEnv = fs.String("env", "", "Only process this environment (folder name under -envdir).")
		envDir    = fs.String("envdir", "../env", "Base directory containing environment folders.")
		outputDir = fs.String("output", "manifests", "Output directory for rendered charts.")
		verbose   = fs.Bool("v", false, "Enable verbose logging.")
	)	

	fs.Usage = func() {
		fmt.Println("Usage: run-manifest-checks render-only [flags]")
		fmt.Println("")
		fmt.Println("Renders all charts found in the ApplicationSets in the specified environment and outputs the manifests to the specified output directory.")
		fmt.Println("")		
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	verboseLogging = *verbose

	if err := runAllChartRenders(*singleEnv, *envDir, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error running chart renders: %v\n", err)
		os.Exit(1)
	}

}


func runAllChartRenders(singleEnv, envDir, outputDir string) error {
	fmt.Println("Starting chart renders...")
	params, err := findChartsInAppsets(envDir, singleEnv)
	if err != nil {
		return fmt.Errorf("failed to find charts in ApplicationSets: %w", err)
	}
	
	fmt.Printf("Found %d charts to process.\n", len(params))

	context := context.Background()

	// Delete output dir if it exists
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to clear output directory: %w", err)
	}

	renderer := ChartRenderingEngine{
		context:    context,
		executor:   &RealCommandExecutor{},
		outputDir:  outputDir,
		inputChan:  make(chan ChartRenderParams),
		resultChan: make(chan RenderResult),
		name:       "ChartRenderer",
		errorChan: make(chan ErrorResult),
		workerWaitGroup: sync.WaitGroup{},
	}
	renderer.Start(10)

	go func() {
		for _, p := range params {
			renderer.inputChan <- p
		}
		close(renderer.inputChan)
	}()

	busy := true
	for busy {
		select {
		case renderResult, ok := <-renderer.resultChan:
			if !ok {
				fmt.Println("No more render results.")
				busy = false
			}
			fmt.Printf(">>> chart %s %s from env %s: ✓ Rendered successfully to %s\n", renderResult.Chart.ChartName, renderResult.Chart.ChartVersion, renderResult.Chart.Env, renderResult.ManifestPath)
		case renderErr := <-renderer.errorChan:
			fmt.Printf(">>> chart %s %s from env %s: ✗ Error: %v\n", renderErr.Chart.ChartName, renderErr.Chart.ChartVersion, renderErr.Chart.Env, renderErr.Error)
		}
	}
	fmt.Printf("Done")
	return nil
}

func runAllChartChecks(singleEnv, envDir, outputDir string) error {
	fmt.Println("Starting chart checks...")
	params, err := findChartsInAppsets(envDir, singleEnv)
	if err != nil {
		return fmt.Errorf("failed to find charts in ApplicationSets: %w", err)
	}
	
	fmt.Printf("Found %d charts to process.\n", len(params))

	context := context.Background()

	// Delete output dir if it exists
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to clear output directory: %w", err)
	}

	appChecker := NewAppCheckerEngine(context, outputDir)
	appChecker.Start(10)

	go func() {
		for _, p := range params {
			appChecker.inputChan <- AppCheckInstruction{Chart: p}
		}
		close(appChecker.inputChan)
	}()

	success := true

	for result := range appChecker.resultChan {
		if result.Error != nil {
			fmt.Printf(">>> chart %s %s from env %s with image %s: ✗ Error: %v\n", result.Chart.ChartName, result.Chart.ChartVersion, result.Chart.Env, result.Image, result.Error)
			success = false
		} else {
			fmt.Printf(">>> chart %s %s from env %s with image %s: ✓ All checks passed\n", result.Chart.ChartName, result.Chart.ChartVersion, result.Chart.Env, result.Image)
		}
	}

	if success {
		fmt.Println("All chart checks completed successfully.")
		return nil
	} else {
		fmt.Println("Some chart checks failed. See above for details.")
		return fmt.Errorf("one or more chart checks failed")
	}
}