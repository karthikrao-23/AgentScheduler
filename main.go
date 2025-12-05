package main

import (
	"agent-scheduler/formatter"
	"agent-scheduler/parser"
	"agent-scheduler/scheduler"
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define flags
	input := flag.String("input", "", "Input CSV file (required)")
	format := flag.String("format", "text", "Output format: text|json|csv")
	utilization := flag.Float64("utilization", 1.0, "Utilization multiplier (between 0 and 1)")
	capacity := flag.Int("capacity", 0, "Maximum agent capacity per hour (0 = unlimited)")

	// Parse command-line flags
	flag.Parse()

	// Validate required input flag
	if *input == "" {
		fmt.Println("Error: -input flag is required")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate format enum
	validFormats := map[string]bool{"text": true, "json": true, "csv": true}
	if !validFormats[*format] {
		fmt.Printf("Error: format must be one of: text, json, csv (got: %s)\n", *format)
		os.Exit(1)
	}

	// Validate utilization range
	if *utilization < 0 || *utilization > 1 {
		fmt.Println("Error: utilization must be between 0 and 1")
		os.Exit(1)
	}

	// Open input file
	file, err := os.Open(*input)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	data, err := parser.Parse(file)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Pass utilization and format to scheduler
	schedule := scheduler.GenerateSchedule(data, *utilization, *capacity)

	// Output based on format
	switch *format {
	case "json":
		fmt.Print(formatter.FormatJSON(schedule))
	case "csv":
		fmt.Print(formatter.FormatCSV(schedule))
	default: // "text"
		fmt.Print(formatter.FormatText(schedule))
	}
}
