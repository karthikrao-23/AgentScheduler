package formatter_test

import (
	"agent-scheduler/formatter"
	"agent-scheduler/models"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatText(t *testing.T) {
	tests := map[string]struct {
		schedule *models.Schedule
		contains []string
	}{
		"EmptySchedule": {
			schedule: &models.Schedule{
				HourlyRequirements: make([][]models.CustomerRequirement, 24),
			},
			contains: []string{
				"00:00 : total=0 ; none",
				"12:00 : total=0 ; none",
				"23:00 : total=0 ; none",
			},
		},
		"SimpleSchedule": {
			schedule: &models.Schedule{
				HourlyRequirements: func() [][]models.CustomerRequirement {
					reqs := make([][]models.CustomerRequirement, 24)
					reqs[10] = []models.CustomerRequirement{
						{Name: "Cust1", AgentsNeeded: 5, Location: time.UTC},
					}
					return reqs
				}(),
			},
			contains: []string{
				"10:00 : total=5 ; [UTC: total=5, Cust1=5]",
			},
		},
		"WithUnmetDemand": {
			schedule: &models.Schedule{
				HourlyRequirements: func() [][]models.CustomerRequirement {
					reqs := make([][]models.CustomerRequirement, 24)
					reqs[10] = []models.CustomerRequirement{
						{Name: "Cust1", AgentsNeeded: 5, Location: time.UTC},
					}
					return reqs
				}(),
				UnmetDemands: []models.UnmetDemand{
					{
						Hour:            10,
						TotalDemand:     10,
						AllocatedAgents: 5,
						UnmetAgents:     5,
						ImpactedClients: []models.ImpactedClient{
							{Name: "Cust2", RequestedAgents: 5, AllocatedAgents: 0, UnmetAgents: 5, Priority: 2},
						},
					},
				},
			},
			contains: []string{
				"10:00 : total=5 ; [UTC: total=5, Cust1=5]",
				"⚠️  CAPACITY WARNING: Demand=10, Allocated=5, Unmet=5",
				"Impacted clients:",
				"• Cust2 [Priority 2]: Requested=5, Allocated=0, Unmet=5",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output := formatter.FormatText(tt.schedule)
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestFormatJSON(t *testing.T) {
	tests := map[string]struct {
		schedule *models.Schedule
		contains []string
	}{
		"EmptySchedule": {
			schedule: &models.Schedule{
				HourlyRequirements: make([][]models.CustomerRequirement, 24),
			},
			contains: []string{
				`"hour": 0`,
				`"total": 0`,
			},
		},
		"SimpleSchedule": {
			schedule: &models.Schedule{
				HourlyRequirements: func() [][]models.CustomerRequirement {
					reqs := make([][]models.CustomerRequirement, 24)
					reqs[10] = []models.CustomerRequirement{
						{Name: "Cust1", AgentsNeeded: 5, Location: time.UTC},
					}
					return reqs
				}(),
			},
			contains: []string{
				`"hour": 10`,
				`"total": 5`,
				`"UTC"`,
				`"Cust1": 5`,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output := formatter.FormatJSON(tt.schedule)
			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}

func TestFormatCSV(t *testing.T) {
	tests := map[string]struct {
		schedule *models.Schedule
		contains []string
	}{
		"EmptySchedule": {
			schedule: &models.Schedule{
				HourlyRequirements: make([][]models.CustomerRequirement, 24),
			},
			contains: []string{
				"Hour,Total Agents,Locations,Customer Details,Capacity Warning,Total Demand,Allocated,Unmet,Impacted Clients",
				"00:00,0,,,No,,,,",
			},
		},
		"SimpleSchedule": {
			schedule: &models.Schedule{
				HourlyRequirements: func() [][]models.CustomerRequirement {
					reqs := make([][]models.CustomerRequirement, 24)
					reqs[10] = []models.CustomerRequirement{
						{Name: "Cust1", AgentsNeeded: 5, Location: time.UTC},
					}
					return reqs
				}(),
			},
			contains: []string{
				"10:00,5,UTC,\"Cust1(UTC,agents=5)\",No,,,,",
			},
		},
		"WithUnmetDemand": {
			schedule: &models.Schedule{
				HourlyRequirements: func() [][]models.CustomerRequirement {
					reqs := make([][]models.CustomerRequirement, 24)
					reqs[10] = []models.CustomerRequirement{
						{Name: "Cust1", AgentsNeeded: 5, Location: time.UTC},
					}
					return reqs
				}(),
				UnmetDemands: []models.UnmetDemand{
					{
						Hour:            10,
						TotalDemand:     10,
						AllocatedAgents: 5,
						UnmetAgents:     5,
						ImpactedClients: []models.ImpactedClient{
							{Name: "Cust2", RequestedAgents: 5, AllocatedAgents: 0, UnmetAgents: 5, Priority: 2},
						},
					},
				},
			},
			contains: []string{
				"10:00,5,UTC,\"Cust1(UTC,agents=5)\",Yes,10,5,5,\"Cust2(priority=2,requested=5,allocated=0,unmet=5)\"",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output := formatter.FormatCSV(tt.schedule)
			lines := strings.Split(output, "\n")

			// Check header
			assert.Equal(t, "Hour,Total Agents,Locations,Customer Details,Capacity Warning,Total Demand,Allocated,Unmet,Impacted Clients", lines[0])

			for _, s := range tt.contains {
				assert.Contains(t, output, s)
			}
		})
	}
}
