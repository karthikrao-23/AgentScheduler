package formatter

import (
	"agent-scheduler/models"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ScheduleData holds prepared schedule data used by all formatters
type ScheduleData struct {
	Hours       []HourlyData
	UnmetByHour map[int]*models.UnmetDemand
}

// HourlyData groups requirements by location for an hour
type HourlyData struct {
	Hour         int                       `json:"hour"`
	Total        int                       `json:"total"`
	LocationData map[string]*LocationGroup `json:"locations,omitempty"`
	UnmetDemand  *UnmetDemandInfo          `json:"unmet_demand,omitempty"`
}

// UnmetDemandInfo represents unmet demand for a specific hour
type UnmetDemandInfo struct {
	TotalDemand     int                     `json:"total_demand"`
	AllocatedAgents int                     `json:"allocated_agents"`
	UnmetAgents     int                     `json:"unmet_agents"`
	ImpactedClients []models.ImpactedClient `json:"impacted_clients"`
}

// LocationGroup holds customer data for a location
type LocationGroup struct {
	Total     int            `json:"total"`
	Customers map[string]int `json:"customers"`
}

// prepareScheduleData extracts and organizes schedule data for formatting
func prepareScheduleData(schedule *models.Schedule) *ScheduleData {
	// Create unmet demand lookup map
	unmetByHour := make(map[int]*models.UnmetDemand)
	for i := range schedule.UnmetDemands {
		unmetByHour[schedule.UnmetDemands[i].Hour] = &schedule.UnmetDemands[i]
	}

	// Process all hours
	hours := make([]HourlyData, 24)
	for h := range 24 {
		hours[h] = processHour(schedule, h)

		// Add unmet demand info if exists
		if unmet, exists := unmetByHour[h]; exists {
			clients := make([]models.ImpactedClient, len(unmet.ImpactedClients))
			for j, client := range unmet.ImpactedClients {
				clients[j] = models.ImpactedClient{
					Name:            client.Name,
					RequestedAgents: client.RequestedAgents,
					AllocatedAgents: client.AllocatedAgents,
					UnmetAgents:     client.UnmetAgents,
					Priority:        client.Priority,
				}
			}
			hours[h].UnmetDemand = &UnmetDemandInfo{
				TotalDemand:     unmet.TotalDemand,
				AllocatedAgents: unmet.AllocatedAgents,
				UnmetAgents:     unmet.UnmetAgents,
				ImpactedClients: clients,
			}
		}
	}

	return &ScheduleData{
		Hours:       hours,
		UnmetByHour: unmetByHour,
	}
}

// FormatText returns the text representation of the schedule
func FormatText(schedule *models.Schedule) string {
	data := prepareScheduleData(schedule)
	var sb strings.Builder

	for _, hourData := range data.Hours {
		sb.WriteString(formatTextLine(hourData.Hour, hourData))
		sb.WriteString("\n")

		// Add unmet demand warning if exists
		if hourData.UnmetDemand != nil {
			unmet := hourData.UnmetDemand
			sb.WriteString(fmt.Sprintf("  ⚠️  CAPACITY WARNING: Demand=%d, Allocated=%d, Unmet=%d\n",
				unmet.TotalDemand, unmet.AllocatedAgents, unmet.UnmetAgents))
			sb.WriteString("  Impacted clients:\n")
			for _, client := range unmet.ImpactedClients {
				sb.WriteString(fmt.Sprintf("    • %s [Priority %d]: Requested=%d, Allocated=%d, Unmet=%d\n",
					client.Name, client.Priority, client.RequestedAgents,
					client.AllocatedAgents, client.UnmetAgents))
			}
		}
	}

	return sb.String()
}

// FormatJSON returns the JSON representation of the schedule
func FormatJSON(schedule *models.Schedule) string {
	data := prepareScheduleData(schedule)
	jsonBytes, _ := json.MarshalIndent(data.Hours, "", "  ")
	return string(jsonBytes)
}

// FormatCSV returns the CSV representation of the schedule
func FormatCSV(schedule *models.Schedule) string {
	data := prepareScheduleData(schedule)
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Write header
	writer.Write([]string{
		"Hour", "Total Agents", "Locations", "Customer Details",
		"Capacity Warning", "Total Demand", "Allocated", "Unmet", "Impacted Clients",
	})

	for _, hourData := range data.Hours {
		writeHourToCSV(writer, hourData)
	}

	writer.Flush()
	return sb.String()
}

// writeHourToCSV writes a single hour's data to CSV
func writeHourToCSV(writer *csv.Writer, hourData HourlyData) {
	hour := hourData.Hour
	unmet := hourData.UnmetDemand

	if hourData.Total == 0 {
		// Empty hour
		writer.Write([]string{
			fmt.Sprintf("%02d:00", hour), "0", "", "",
			"No", "", "", "", "",
		})
		return
	}

	// Build location list
	locations := getSortedLocations(hourData.LocationData)
	locationList := strings.Join(locations, "; ")

	// Build customer details with format: "Customer1(loc1,agents=5); Customer2(loc2,agents=3)"
	var customerDetails []string
	for _, loc := range locations {
		locData := hourData.LocationData[loc]
		customers := getSortedCustomers(locData.Customers)

		for _, customer := range customers {
			agents := locData.Customers[customer]
			customerDetails = append(customerDetails,
				fmt.Sprintf("%s(%s,agents=%d)", customer, loc, agents))
		}
	}
	customerDetailsStr := strings.Join(customerDetails, "; ")

	// Build impacted clients string
	var impactedClientsStr string
	if unmet != nil {
		var impactedParts []string
		for _, client := range unmet.ImpactedClients {
			impactedParts = append(impactedParts,
				fmt.Sprintf("%s(priority=%d,requested=%d,allocated=%d,unmet=%d)",
					client.Name, client.Priority, client.RequestedAgents,
					client.AllocatedAgents, client.UnmetAgents))
		}
		impactedClientsStr = strings.Join(impactedParts, "; ")
	}

	// Build single row for this hour
	row := []string{
		fmt.Sprintf("%02d:00", hour),
		fmt.Sprintf("%d", hourData.Total),
		locationList,
		customerDetailsStr,
	}

	if unmet != nil {
		row = append(row,
			"Yes",
			fmt.Sprintf("%d", unmet.TotalDemand),
			fmt.Sprintf("%d", unmet.AllocatedAgents),
			fmt.Sprintf("%d", unmet.UnmetAgents),
			impactedClientsStr,
		)
	} else {
		row = append(row, "No", "", "", "", "")
	}

	writer.Write(row)
}

// processHour groups requirements by location for a given hour
func processHour(schedule *models.Schedule, hour int) HourlyData {
	data := HourlyData{
		Hour:         hour,
		LocationData: make(map[string]*LocationGroup),
	}

	if hour >= len(schedule.HourlyRequirements) {
		return data
	}

	requirements := schedule.HourlyRequirements[hour]

	for _, req := range requirements {
		locName := req.Location.String()

		if _, exists := data.LocationData[locName]; !exists {
			data.LocationData[locName] = &LocationGroup{
				Customers: make(map[string]int),
			}
		}

		data.LocationData[locName].Customers[req.Name] = req.AgentsNeeded
		data.LocationData[locName].Total += req.AgentsNeeded
		data.Total += req.AgentsNeeded
	}

	return data
}

// formatTextLine formats a single hour line for text output
func formatTextLine(hour int, data HourlyData) string {
	if data.Total == 0 {
		return fmt.Sprintf("%02d:00 : total=0 ; none", hour)
	}

	var parts []string
	locations := getSortedLocations(data.LocationData)

	for _, loc := range locations {
		locData := data.LocationData[loc]
		var locParts []string
		locParts = append(locParts, fmt.Sprintf("total=%d", locData.Total))

		customers := getSortedCustomers(locData.Customers)
		for _, customer := range customers {
			locParts = append(locParts, fmt.Sprintf("%s=%d", customer, locData.Customers[customer]))
		}

		parts = append(parts, fmt.Sprintf("%s: %s", loc, strings.Join(locParts, ", ")))
	}

	return fmt.Sprintf("%02d:00 : total=%d ; [%s]", hour, data.Total, strings.Join(parts, ", "))
}

// getSortedLocations returns sorted location names
func getSortedLocations(locationData map[string]*LocationGroup) []string {
	locations := make([]string, 0, len(locationData))
	for loc := range locationData {
		locations = append(locations, loc)
	}
	sort.Strings(locations)
	return locations
}

// getSortedCustomers returns sorted customer names
func getSortedCustomers(customers map[string]int) []string {
	names := make([]string, 0, len(customers))
	for name := range customers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
