package scheduler

import (
	"agent-scheduler/models"
	"math"
	"sort"
	"time"
)

// GenerateSchedule calculates the number of agents needed per hour for each customer.
func GenerateSchedule(data []models.CallData, utilization float64, capacityPerHour int) *models.Schedule {
	hourlyRequests := make([][]models.CustomerRequirement, 24)
	for h := range 24 {
		hourlyRequests[h] = make([]models.CustomerRequirement, 0)
	}

	for _, cd := range data {
		start := cd.StartTime
		end := cd.EndTime

		// Handle overnight shifts (e.g., 9PM to 5AM)
		if end.Before(start) {
			end = end.Add(24 * time.Hour)
		}

		// Find the elapsed duration in hours and not use wall clock to
		// account for DST.
		durationHours := end.Sub(start).Hours()
		if durationHours <= 0 {
			continue
		}

		callsPerHour := float64(cd.NumberOfCalls) / durationHours

		// Determine the hour boundaries to schedule
		// Round start down to hour boundary, round end up to hour boundary
		startHourBoundary := time.Date(start.Year(), start.Month(), start.Day(),
			start.Hour(), 0, 0, 0, start.Location())
		endHourBoundary := time.Date(end.Year(), end.Month(), end.Day(),
			end.Hour(), 0, 0, 0, end.Location())

		// If end time has minutes/seconds, we need to include that hour too
		if end.After(endHourBoundary) {
			endHourBoundary = endHourBoundary.Add(time.Hour)
		}

		// Iterate hour by hour at hourly boundaries
		for t := startHourBoundary; t.Before(endHourBoundary); t = t.Add(time.Hour) {
			// Calculate the fraction of this hour that's actually being used
			hourStart := t
			hourEnd := t.Add(time.Hour)

			// Clamp to actual work window
			actualStart := hourStart
			if start.After(hourStart) {
				actualStart = start
			}
			actualEnd := hourEnd
			if end.Before(hourEnd) {
				actualEnd = end
			}

			// Calculate fraction of hour being used
			hoursUsedInThisSlot := actualEnd.Sub(actualStart).Hours()
			if hoursUsedInThisSlot <= 0 {
				continue
			}

			// Calls in this specific hour slot based on fraction
			callsThisHour := callsPerHour * hoursUsedInThisSlot

			// Agents = ceil(calls_this_hour * avg_duration / 3600)
			agentsNeeded := int(math.Ceil(callsThisHour * float64(cd.AverageCallDurationSeconds) / 3600.0))

			// Adjust agents needed based on utilization
			utilizationMultiplier := 1 / utilization
			agentsNeeded = int(math.Ceil(float64(agentsNeeded) * utilizationMultiplier))

			localTime := t
			if cd.Location != nil {
				localTime = t.In(cd.Location)
			}
			h := localTime.Hour()
			hourlyRequests[h] = append(
				hourlyRequests[h], models.CustomerRequirement{
					Name:         cd.CustomerName,
					AgentsNeeded: agentsNeeded,
					Location:     cd.Location,
					Priority:     cd.Priority,
				},
			)
		}
	}

	schedule := models.Schedule{
		HourlyRequirements: hourlyRequests,
		UnmetDemands:       make([]models.UnmetDemand, 0),
	}
	// Apply capacity constraints if capacityPerHour > 0
	if capacityPerHour > 0 {
		for h := range 24 {
			allocated, unmet := allocateWithConstraints(hourlyRequests[h], capacityPerHour)
			schedule.HourlyRequirements[h] = allocated
			if unmet != nil {
				unmet.Hour = h
				schedule.UnmetDemands = append(schedule.UnmetDemands, *unmet)
			}
		}
	}

	return &schedule
}

// allocateWithConstraints performs priority-based allocation.
// Time: O(n log n) for sort + O(n) for allocation = O(n log n)
// Space: O(n) for output slices (no extra map overhead)
func allocateWithConstraints(requests []models.CustomerRequirement, capacity int) ([]models.CustomerRequirement, *models.UnmetDemand) {
	if len(requests) == 0 {
		return nil, nil
	}

	// Calculate total demand: O(n)
	totalDemand := 0
	for _, req := range requests {
		totalDemand += req.AgentsNeeded
	}

	// Fast path: if capacity exceeds demand, no allocation logic needed
	if capacity >= totalDemand {
		return requests, nil
	}

	// Sort by priority (1 = highest): O(n log n)
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Priority < requests[j].Priority
	})

	// Pre-allocate with capacity hints to reduce reallocations
	allocated := make([]models.CustomerRequirement, 0, len(requests))
	impactedClients := make([]models.ImpactedClient, 0)
	remaining := capacity

	// Single pass allocation: O(n)
	for _, req := range requests {
		if remaining <= 0 {
			// No capacity left - fully unmet
			impactedClients = append(impactedClients, models.ImpactedClient{
				Name:            req.Name,
				RequestedAgents: req.AgentsNeeded,
				AllocatedAgents: 0,
				UnmetAgents:     req.AgentsNeeded,
				Priority:        req.Priority,
			})
			continue
		}

		if remaining >= req.AgentsNeeded {
			// Full allocation
			allocated = append(allocated, req)
			remaining -= req.AgentsNeeded
		} else {
			// Partial allocation - give what's left
			allocated = append(allocated, models.CustomerRequirement{
				Name:         req.Name,
				AgentsNeeded: remaining,
				Location:     req.Location,
				Priority:     req.Priority,
			})
			impactedClients = append(impactedClients, models.ImpactedClient{
				Name:            req.Name,
				RequestedAgents: req.AgentsNeeded,
				AllocatedAgents: remaining,
				UnmetAgents:     req.AgentsNeeded - remaining,
				Priority:        req.Priority,
			})
			remaining = 0
		}
	}

	// Only create UnmetDemand if there are impacted clients
	if len(impactedClients) > 0 {
		return allocated, &models.UnmetDemand{
			TotalDemand:     totalDemand,
			AllocatedAgents: capacity,
			UnmetAgents:     totalDemand - capacity,
			ImpactedClients: impactedClients,
		}
	}
	return allocated, nil
}
