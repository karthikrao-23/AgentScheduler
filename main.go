package main

import (
	"agent-scheduler/formatter"
	"agent-scheduler/metrics"
	"agent-scheduler/parser"
	"agent-scheduler/scheduler"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"
)

func main() {
	// Define flags
	input := flag.String("input", "", "Input CSV file (required)")
	format := flag.String("format", "text", "Output format: text|json|csv")
	utilization := flag.Float64("utilization", 1.0, "Utilization multiplier (between 0 and 1)")
	capacity := flag.Int("capacity", 0, "Maximum agent capacity per hour (0 = unlimited)")
	metricsAddr := flag.String("metrics-addr", "", "Address to expose Prometheus metrics (e.g., :9090)")
	pushGateway := flag.String("push-url", "", "Pushgateway URL to push metrics to (e.g., http://localhost:9091)")
	wait := flag.Bool("wait", false, "Keep process running after completion to allow for metric scraping")

	// Parse command-line flags
	flag.Parse()

	// Start metrics server if address provided
	if *metricsAddr != "" {
		go func() {
			http.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))
			fmt.Printf("Metrics server listening on %s/metrics\n", *metricsAddr)
			if err := http.ListenAndServe(*metricsAddr, nil); err != nil {
				fmt.Printf("Metrics server error: %v\n", err)
			}
		}()
	}

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

	// Handle metrics pushing or waiting
	if *pushGateway != "" {
		jobName := "agent_scheduler"
		if err := push.New(*pushGateway, jobName).Gatherer(metrics.Registry).Push(); err != nil {
			fmt.Fprintf(os.Stderr, "Error pushing to Pushgateway: %v\n", err)
		} else {
			fmt.Println("\nMetrics successfully pushed to Pushgateway")
		}
	}

	if *wait && *metricsAddr != "" {
		fmt.Println("\nProcess kept alive for metric scraping. Press Ctrl+C to exit.")
		// Wait for interrupt signal
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		fmt.Println("\nExiting...")
	} else if *metricsAddr != "" && *pushGateway == "" {
		// Small delay to allow final scrape if not waiting explicitly
		// but typically batch jobs should use pushgateway or wait
		time.Sleep(100 * time.Millisecond)
	}
}
