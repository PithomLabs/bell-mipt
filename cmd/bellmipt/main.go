package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/PithomLabs/bell-mipt/internal/bellmipt"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config (default: built-in)")
	outDir := flag.String("out", "out/bellmipt-run", "output directory")
	flag.Parse()

	var cfg bellmipt.Config
	if *configPath != "" {
		var err error
		cfg, err = bellmipt.LoadConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg = bellmipt.DefaultConfig()
	}

	result := bellmipt.Run(cfg, *outDir)
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error running simulation: %v\n", result.Error)
		os.Exit(1)
	}

	os.Exit(bellmipt.ExitCodeForGoalStatus(result.Report.GoalStatus))
}
