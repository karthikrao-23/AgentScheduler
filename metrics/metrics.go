// Package metrics provides Prometheus observability metrics for the agent scheduler.
// It includes Critical and Important metrics for business and operational visibility.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registry is the custom prometheus registry for our application
var Registry = prometheus.NewRegistry()

// factory allows us to register metrics to our custom Registry directly
var factory = promauto.With(Registry)

// =============================================================================
// CRITICAL METRICS - Business Impact Visibility
// =============================================================================

// AgentsUnmetTotal tracks total unmet agent demand across all hours.
// High values indicate capacity planning issues.
var AgentsUnmetTotal = factory.NewGauge(prometheus.GaugeOpts{
	Namespace: "scheduler",
	Name:      "agents_unmet_total",
	Help:      "Total number of agents that could not be allocated due to capacity constraints",
})

// AgentsDemandedTotal tracks total agent demand across all hours.
var AgentsDemandedTotal = factory.NewGauge(prometheus.GaugeOpts{
	Namespace: "scheduler",
	Name:      "agents_demanded_total",
	Help:      "Total number of agents demanded across all customers and hours",
})

// AgentsAllocatedTotal tracks total agents successfully allocated.
var AgentsAllocatedTotal = factory.NewGauge(prometheus.GaugeOpts{
	Namespace: "scheduler",
	Name:      "agents_allocated_total",
	Help:      "Total number of agents successfully allocated",
})

// HighPriorityFullySatisfied tracks count of priority-1 requests fully satisfied.
var HighPriorityFullySatisfied = factory.NewCounter(prometheus.CounterOpts{
	Namespace: "scheduler",
	Name:      "high_priority_fully_satisfied_total",
	Help:      "Count of priority-1 (highest) requests that were fully satisfied",
})

// HighPriorityPartiallySatisfied tracks count of priority-1 requests only partially satisfied.
var HighPriorityPartiallySatisfied = factory.NewCounter(prometheus.CounterOpts{
	Namespace: "scheduler",
	Name:      "high_priority_partially_satisfied_total",
	Help:      "Count of priority-1 requests that were only partially satisfied",
})

// HighPriorityUnsatisfied tracks count of priority-1 requests with zero allocation.
var HighPriorityUnsatisfied = factory.NewCounter(prometheus.CounterOpts{
	Namespace: "scheduler",
	Name:      "high_priority_unsatisfied_total",
	Help:      "Count of priority-1 requests that received zero allocation",
})

// HoursWithUnmetDemand tracks number of hours where capacity was exceeded.
var HoursWithUnmetDemand = factory.NewGauge(prometheus.GaugeOpts{
	Namespace: "scheduler",
	Name:      "hours_with_unmet_demand",
	Help:      "Number of hours in the schedule where demand exceeded capacity",
})

// UnmetDemandByPriority tracks unmet agents by priority level.
var UnmetDemandByPriority = factory.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "scheduler",
	Name:      "unmet_demand_by_priority",
	Help:      "Unmet agent demand broken down by priority level",
}, []string{"priority"})

// =============================================================================
// IMPORTANT METRICS - Operational Health
// =============================================================================

// ParserErrorsTotal tracks parse errors by error type.
var ParserErrorsTotal = factory.NewCounterVec(prometheus.CounterOpts{
	Namespace: "parser",
	Name:      "errors_total",
	Help:      "Total parse errors by error type",
}, []string{"error_type"})

// ParserRecordsTotal tracks total records successfully parsed.
var ParserRecordsTotal = factory.NewCounter(prometheus.CounterOpts{
	Namespace: "parser",
	Name:      "records_total",
	Help:      "Total CSV records successfully parsed",
})

// ParserDurationSeconds tracks time to parse input files.
var ParserDurationSeconds = factory.NewHistogram(prometheus.HistogramOpts{
	Namespace: "parser",
	Name:      "duration_seconds",
	Help:      "Time taken to parse CSV input file",
	Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
})

// SchedulerDurationSeconds tracks time to generate schedule.
var SchedulerDurationSeconds = factory.NewHistogram(prometheus.HistogramOpts{
	Namespace: "scheduler",
	Name:      "duration_seconds",
	Help:      "Time taken to generate the schedule",
	Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
})

// SchedulerCustomersProcessed tracks number of customers per scheduling run.
var SchedulerCustomersProcessed = factory.NewHistogram(prometheus.HistogramOpts{
	Namespace: "scheduler",
	Name:      "customers_processed",
	Help:      "Number of customers processed per scheduling run",
	Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
})

// SchedulerCapacityUsed tracks the capacity used when constraints are applied.
var SchedulerCapacityUsed = factory.NewGauge(prometheus.GaugeOpts{
	Namespace: "scheduler",
	Name:      "capacity_used_total",
	Help:      "Total capacity used across all hours when capacity constraints applied",
})

// =============================================================================
// Helper Functions
// =============================================================================

// ResetSchedulerGauges resets all scheduler gauges before a new scheduling run.
// Call this at the start of GenerateSchedule.
func ResetSchedulerGauges() {
	AgentsUnmetTotal.Set(0)
	AgentsDemandedTotal.Set(0)
	AgentsAllocatedTotal.Set(0)
	HoursWithUnmetDemand.Set(0)
	SchedulerCapacityUsed.Set(0)
	UnmetDemandByPriority.Reset()
}
