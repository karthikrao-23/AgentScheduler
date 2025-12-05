package scheduler_test

import (
	"agent-scheduler/models"
	"agent-scheduler/scheduler"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSchedule(t *testing.T) {
	// Helper to create time in a specific location
	makeTime := func(hour int, locName string) time.Time {
		loc, err := time.LoadLocation(locName)
		if err != nil {
			panic(err)
		}
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, loc)
	}

	// Helper to create time with minutes in a specific location
	makeTimeWithMinutes := func(hour, minute int, locName string) time.Time {
		loc, err := time.LoadLocation(locName)
		if err != nil {
			panic(err)
		}
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	}

	// Helper to load a location
	mustLoadLocation := func(locName string) *time.Location {
		loc, err := time.LoadLocation(locName)
		if err != nil {
			panic(err)
		}
		return loc
	}

	tests := map[string]struct {
		input    []models.CallData
		expected map[int]int // map[hourLocal]totalAgents (in customer's local time)
	}{
		"Simple_SameDay_UTC": {
			input: []models.CallData{
				{
					CustomerName:               "Cust1",
					AverageCallDurationSeconds: 3600,                // 1 hour
					StartTime:                  makeTime(10, "UTC"), // 10:00 UTC
					EndTime:                    makeTime(12, "UTC"), // 12:00 UTC
					Location:                   time.UTC,
					NumberOfCalls:              10,
					Priority:                   1,
				},
			},
			// Duration = 2 hours. Calls/hr = 5.
			// Agents = ceil(5 * 3600 / 3600) = 5.
			// Hours (local UTC): 10, 11.
			expected: map[int]int{
				10: 5,
				11: 5,
			},
		},
		"Overnight_PST": {
			input: []models.CallData{
				{
					CustomerName:               "Cust2",
					AverageCallDurationSeconds: 3600,
					StartTime:                  makeTime(22, "America/Los_Angeles"), // 10 PM PST
					EndTime:                    makeTime(2, "America/Los_Angeles"),  // 2 AM PST (next day)
					Location:                   mustLoadLocation("America/Los_Angeles"),
					NumberOfCalls:              20,
					Priority:                   1,
				},
			},
			// Duration = 4 hours. Calls/hr = 5. Agents = 5.
			// Hours (local PST): 22, 23, 0, 1.
			expected: map[int]int{
				22: 5,
				23: 5,
				0:  5,
				1:  5,
			},
		},
		"Mixed_Timezones": {
			input: []models.CallData{
				{
					CustomerName:               "CustPST",
					AverageCallDurationSeconds: 3600,
					StartTime:                  makeTime(9, "America/Los_Angeles"),  // 9 AM PST
					EndTime:                    makeTime(11, "America/Los_Angeles"), // 11 AM PST
					Location:                   mustLoadLocation("America/Los_Angeles"),
					NumberOfCalls:              10,
				},
				{
					CustomerName:               "CustEST",
					AverageCallDurationSeconds: 3600,
					StartTime:                  makeTime(12, "America/New_York"), // 12 PM EST
					EndTime:                    makeTime(14, "America/New_York"), // 2 PM EST
					Location:                   mustLoadLocation("America/New_York"),
					NumberOfCalls:              10,
				},
			},
			// CustPST: local hours 9, 10 (5 agents each)
			// CustEST: local hours 12, 13 (5 agents each)
			// These are different local hours, so no overlap
			expected: map[int]int{
				9:  5,
				10: 5,
				12: 5,
				13: 5,
			},
		},
		"NonHourBoundary_PartialHours": {
			input: []models.CallData{
				{
					CustomerName:               "PartialHourCustomer",
					AverageCallDurationSeconds: 1800,                                            // 30 minutes per call
					StartTime:                  makeTimeWithMinutes(9, 30, "America/New_York"),  // 9:30 AM EST
					EndTime:                    makeTimeWithMinutes(15, 45, "America/New_York"), // 3:45 PM EST
					Location:                   mustLoadLocation("America/New_York"),
					NumberOfCalls:              100,
					Priority:                   1,
				},
			},
			// This means:
			// - Hour 9: covers 9:30-10:00 (partial hour - 30 min)
			// - Hours 10-14: full hours within the time window
			// - Hour 15: covers 15:00-15:45 (partial hour - 45 min)
			expected: map[int]int{
				9:  4,
				10: 8,
				11: 8,
				12: 8,
				13: 8,
				14: 8,
				15: 6,
			},
		},
		"DST_SpringForward": {
			input: []models.CallData{
				{
					CustomerName:               "SpringForwardTest",
					AverageCallDurationSeconds: 3600,
					// March 10, 2024 - DST spring forward: 1:00 AM to 4:00 AM
					// Actual elapsed time: 2 hours (hour 2:00-2:59 doesn't exist)
					StartTime:     time.Date(2024, 3, 10, 1, 0, 0, 0, mustLoadLocation("America/New_York")),
					EndTime:       time.Date(2024, 3, 10, 4, 0, 0, 0, mustLoadLocation("America/New_York")),
					Location:      mustLoadLocation("America/New_York"),
					NumberOfCalls: 6, // 6 calls / 2 hours = 3 calls/hour
					Priority:      1,
				},
			},
			// Calls per hour = 3, Agents per hour = 3
			// Hours scheduled: 1, 3 (hour 2 doesn't exist due to DST)
			expected: map[int]int{
				1: 3, // 1:00-3:00 (skips to 3:00 due to DST)
				3: 3, // 3:00-4:00
			},
		},
		"DST_FallBack": {
			input: []models.CallData{
				{
					CustomerName:               "FallBackTest",
					AverageCallDurationSeconds: 3600,
					// November 3, 2024 - DST fall back: 12:00 AM to 3:00 AM
					// Actual elapsed time: 4 hours (hour 1:00-1:59 repeats)
					StartTime:     time.Date(2024, 11, 3, 0, 0, 0, 0, mustLoadLocation("America/New_York")),
					EndTime:       time.Date(2024, 11, 3, 3, 0, 0, 0, mustLoadLocation("America/New_York")),
					Location:      mustLoadLocation("America/New_York"),
					NumberOfCalls: 12, // 12 calls / 4 hours = 3 calls/hour
					Priority:      1,
				},
			},
			// Calls per hour = 3, Agents per hour = 3
			// Hours scheduled: 0, 1, 2
			// Hour 1 occurs twice during fall back, so gets 2 hours worth of agents
			expected: map[int]int{
				0: 3, // 0:00-1:00
				1: 6, // 1:00-2:00 (happens twice = 2 hours × 3 agents/hour)
				2: 3, // 2:00-3:00
			},
		},
		"DST_PartialHoursAcrossSpringForward": {
			input: []models.CallData{
				{
					CustomerName:               "PartialDSTTest",
					AverageCallDurationSeconds: 1800, // 30 min per call
					// March 10, 2024: 1:30 AM to 3:30 AM (spans DST)
					// Actual elapsed time: 1 hour (not 2!)
					StartTime:     time.Date(2024, 3, 10, 1, 30, 0, 0, mustLoadLocation("America/New_York")),
					EndTime:       time.Date(2024, 3, 10, 3, 30, 0, 0, mustLoadLocation("America/New_York")),
					Location:      mustLoadLocation("America/New_York"),
					NumberOfCalls: 10, // 10 calls / 1 hour = 10 calls/hour
					Priority:      1,
				},
			},
			// Hour 1: 1:30-3:00 (jumps to 3:00, actual 0.5 hours) = 5 calls = 3 agents
			// Hour 3: 3:00-3:30 (0.5 hours) = 5 calls = 3 agents
			expected: map[int]int{
				1: 3, // Partial hour before DST jump
				2: 0, // Hour 2 doesn't exist due to DST
				3: 3, // Partial hour after DST jump
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sched := scheduler.GenerateSchedule(tt.input, 1.0, 0)

			for h, reqs := range sched.HourlyRequirements {
				total := 0
				for _, r := range reqs {
					total += r.AgentsNeeded
				}

				if expectedTotal, ok := tt.expected[h]; ok {
					assert.Equal(t, expectedTotal, total, fmt.Sprintf("Hour %d agents mismatch", h))
				} else {
					assert.Equal(t, 0, total, fmt.Sprintf("Hour %d should be empty", h))
				}
			}
		})
	}
}

func TestGenerateSchedule_PriorityAndCapacity(t *testing.T) {
	// Helper to create time in UTC
	makeTime := func(hour int) time.Time {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	}

	input := []models.CallData{
		{
			CustomerName:               "HighPriority",
			AverageCallDurationSeconds: 3600, // 1 hour
			StartTime:                  makeTime(10),
			EndTime:                    makeTime(11),
			Location:                   time.UTC,
			NumberOfCalls:              10, // Needs 10 agents
			Priority:                   1,  // High priority
		},
		{
			CustomerName:               "LowPriority",
			AverageCallDurationSeconds: 3600, // 1 hour
			StartTime:                  makeTime(10),
			EndTime:                    makeTime(11),
			Location:                   time.UTC,
			NumberOfCalls:              10, // Needs 10 agents
			Priority:                   2,  // Low priority
		},
	}

	// Capacity 15. Total demand 20.
	// HighPriority should get 10.
	// LowPriority should get 5.
	sched := scheduler.GenerateSchedule(input, 1.0, 15)

	// Check Hour 10
	reqs := sched.HourlyRequirements[10]
	assert.NotEmpty(t, reqs, "Hour 10 should have requirements")

	// Verify allocations
	var highPriorityAllocated, lowPriorityAllocated int
	for _, r := range reqs {
		switch r.Name {
		case "HighPriority":
			highPriorityAllocated = r.AgentsNeeded
		case "LowPriority":
			lowPriorityAllocated = r.AgentsNeeded
		default:
			panic("Unexpected customer name")
		}
	}

	assert.Equal(t, 10, highPriorityAllocated, "High priority customer should get full allocation")
	assert.Equal(t, 5, lowPriorityAllocated, "Low priority customer should get remaining capacity")

	// Verify Unmet Demands
	assert.NotEmpty(t, sched.UnmetDemands, "Should have unmet demands")

	var foundUnmet bool
	for _, unmet := range sched.UnmetDemands {
		if unmet.Hour == 10 {
			foundUnmet = true
			assert.Equal(t, 20, unmet.TotalDemand, "Total demand mismatch")
			assert.Equal(t, 15, unmet.AllocatedAgents, "Allocated agents mismatch")
			assert.Equal(t, 5, unmet.UnmetAgents, "Unmet agents mismatch")

			// Check impacted clients
			assert.NotEmpty(t, unmet.ImpactedClients, "Should have impacted clients")
			for _, client := range unmet.ImpactedClients {
				if client.Name == "LowPriority" {
					assert.Equal(t, 10, client.RequestedAgents, "LowPriority requested mismatch")
					assert.Equal(t, 5, client.AllocatedAgents, "LowPriority allocated mismatch")
					assert.Equal(t, 5, client.UnmetAgents, "LowPriority unmet mismatch")
				}
			}
		}
	}
	assert.True(t, foundUnmet, "Should find unmet demand for hour 10")
}

func TestGenerateSchedule_Utilization(t *testing.T) {
	makeTime := func(hour int) time.Time {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	}

	input := []models.CallData{
		{
			CustomerName:               "UtilizationTest",
			AverageCallDurationSeconds: 3600,
			StartTime:                  makeTime(10),
			EndTime:                    makeTime(11),
			Location:                   time.UTC,
			NumberOfCalls:              10,
			Priority:                   1,
		},
	}

	// Base agents needed = 10 (10 calls * 1 hr / 1 hr)
	// Utilization 0.8 -> Multiplier = 1/0.8 = 1.25
	// Expected agents = ceil(10 * 1.25) = 13

	sched := scheduler.GenerateSchedule(input, 0.8, 0)

	reqs := sched.HourlyRequirements[10]
	assert.NotEmpty(t, reqs)
	assert.Equal(t, 13, reqs[0].AgentsNeeded, "Should adjust agents based on utilization")
}
