package models

import "time"

// CallData represents the parsed input data for a customer call batch.
// It is shared across packages to schedule calls.
type CallData struct {
	CustomerName               string
	AverageCallDurationSeconds int
	StartTime                  time.Time
	EndTime                    time.Time
	Location                   *time.Location
	NumberOfCalls              int
	Priority                   int
}

// Schedule represents the agent requirements per hour.
type Schedule struct {
	// HourlyRequirements maps hour (0-23) to a list of customer requirements
	HourlyRequirements [][]CustomerRequirement
	// UnmetDemands tracks hours where capacity was exceeded
	UnmetDemands []UnmetDemand
}

// CustomerRequirement holds the number of agents needed for a specific customer.
type CustomerRequirement struct {
	Name         string
	AgentsNeeded int
	Location     *time.Location
	Priority     int
}

// UnmetDemand tracks when demand cannot be met due to capacity constraints
type UnmetDemand struct {
	Hour            int
	TotalDemand     int
	AllocatedAgents int
	UnmetAgents     int
	ImpactedClients []ImpactedClient
}

// ImpactedClient represents a customer whose demand was not fully met
type ImpactedClient struct {
	Name            string
	RequestedAgents int
	AllocatedAgents int
	UnmetAgents     int
	Priority        int
}
